package services

import (
	"cloud.google.com/go/firestore"
	apiAdmin "cloud.google.com/go/firestore/apiv1/admin"
	"cloud.google.com/go/firestore/apiv1/admin/adminpb"
	"context"
	"eventsync/models"
	"eventsync/utils"
	"fmt"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io"
	"log"
	"strings"
	"time"
)

// EventService handles the event related operation, based on Firestore for persistence layer and a reference to
// the configuration service
type EventService struct {
	firestoreClient *firestore.Client
	configService   *ConfigService
}

// EventPathPrefix is the prefix used by the HTTP handler to expose the path prefix to submit events on endpoints
const EventPathPrefix = "/event/"

// NewEventService creates the Event service. It requires a context to create a FirestoreClient
// instance and to create/check the firestore index to be able to query correctly the firestore collection.
// The configService is provided to store and keep the config in the service.
func NewEventService(ctx context.Context, configService *ConfigService) (event *EventService, err error) {
	event = &EventService{configService: configService}

	projectID := utils.GetProjectId()

	event.firestoreClient, err = firestore.NewClient(ctx, projectID)
	if err != nil {
		fmt.Printf("firestore new client error:%s\n", err)
		return
	}

	//To ensure the correct Firestore collection querying, an index must exist
	err = checkAndCreateIndex(ctx, projectID, configService.GetConfig().ServiceName)
	if err != nil {
		return
	}
	return
}

// checkAndCreateIndex creates the used index in Firestore. If it already exists, nothing is performed.
func checkAndCreateIndex(ctx context.Context, projectID string, configName string) (err error) {

	// Create the Admin client
	adminClient, err := apiAdmin.NewFirestoreAdminClient(ctx)
	if err != nil {
		fmt.Printf("firestore admin new Client error:%s\n", err)
		return err
	}

	// Predefine the default order
	ascendingFieldOrder := adminpb.Index_IndexField_Order_{
		Order: adminpb.Index_IndexField_ASCENDING,
	}

	indexParent := fmt.Sprintf("projects/%s/databases/(default)/collectionGroups/%s", projectID, configName)

	// create the indexes with corresponding fields
	fields := []*adminpb.Index_IndexField{
		{
			FieldPath: "EventKey",
			ValueMode: &ascendingFieldOrder,
		},
		{
			FieldPath: "AlreadyExported",
			ValueMode: &ascendingFieldOrder,
		},
		{
			FieldPath: "Datetime",
			ValueMode: &ascendingFieldOrder,
		},
	}
	operation, err := adminClient.CreateIndex(ctx, &adminpb.CreateIndexRequest{
		Parent: indexParent,
		Index: &adminpb.Index{
			QueryScope: adminpb.Index_COLLECTION,
			Fields:     fields,
		},
	})

	if err != nil && status.Convert(err).Code() == codes.AlreadyExists {
		fmt.Printf("the index already exist. No need to recreate it, the service is fully ready to use\n")
		return nil
	}

	if err != nil {
		log.Fatalf("impossible to create the firestore index on the collection %s because of this error: %s\n", configName, err)
	}

	if operation != nil {
		fmt.Printf("the index has just been created. You have to wait the end of the creation to be able to generate trigger. It can take a few minutes to complete\n")
		return nil
	}

	return nil
}

// FormatEvent takes the raw parts of an HTTP requests and create a models.Event object with those part, without
// transformation.
func FormatEvent(eventKey string, queryParam map[string][]string, headers map[string][]string, body io.ReadCloser) (event models.Event, err error) {
	event.EventKey = eventKey
	event.Datetime = time.Now()

	event.QueryParams = queryParam
	event.Headers = headers

	b, err := io.ReadAll(body)
	if err != nil {
		fmt.Printf("impossible to read the body with err %s\n", err)
		return
	}
	defer body.Close()

	event.Content = string(b)
	return
}

// ExtractEventKey extracts the eventKey value from the URL path in the HTTP request.
// Path must start with the EventPathPrefix
func ExtractEventKey(path string) (eventKey string) {
	// Split according to the EventPathPrefix. the first part should be empty (start with), the 2nd contain the eventKey
	splits := strings.SplitN(path, EventPathPrefix, 2)
	if len(splits) != 2 || strings.Index(path, EventPathPrefix) != 0 {
		return ""
	}
	return splits[1]
}

// StoreEvent persists an event in Firestore. The collection name is the config serviceName value
func (e *EventService) StoreEvent(ctx context.Context, event models.Event) (err error) {
	_, _, err = e.firestoreClient.Collection(e.configService.GetConfig().ServiceName).Add(ctx, event)
	if err != nil {
		return
	}
	fmt.Printf("event correct stored to Firestore collection %s\n", e.configService.GetConfig().ServiceName)
	return
}

// GetEventsOverAPeriod retrieves the events stored in Firestore in the past observationPeriod. Only the not
// alreadyExported event are taken into account. The eventlist groups the event per eventKeys.
func (e *EventService) GetEventsOverAPeriod(ctx context.Context, observationPeriod int64) (eventList map[string]models.EventList, err error) {

	//Define globally the time of reference
	targetDatetime := time.Now().Add(-time.Duration(observationPeriod) * time.Second)

	// The base query select the events in the observation period and not already exported
	query := e.firestoreClient.Collection(e.configService.GetConfig().ServiceName).Where("Datetime", ">", targetDatetime).Where("AlreadyExported", "==", false)

	eventList = make(map[string]models.EventList, len(e.configService.GetConfig().Endpoints))

	//Perform Query for all endpoints in th config
	for _, endpoint := range e.configService.GetConfig().Endpoints {
		// Specialize the query to request per eventKey
		iter := query.Where("EventKey", "==", endpoint.EventKey).Documents(ctx)
		counter := 0

		// Initialize the  list of events of that eventKey
		eventGroup := models.EventList{}
		events := make([]models.Event, 0)
		for {
			var doc *firestore.DocumentSnapshot
			doc, err = iter.Next()
			if err == iterator.Done {
				err = nil
				break
			}
			if err != nil {
				fmt.Printf("error during the document retrieval with error: %s\n", err)
				return
			}
			// Count the number of events in the observation period
			counter += 1
			event := &models.Event{}
			err = doc.DataTo(&event)
			if err != nil {
				fmt.Printf("error during the document conversion with error: %s\n", err)
				return
			}
			// Keep the documentID for later use
			event.FirestoreDocumentID = doc.Ref.ID
			events = append(events, *event)

			//TODO add keep all/first/last here
			// Update the metrics (last and first event in the observation period
			if eventGroup.LastEventDate == nil || event.Datetime.After(*eventGroup.LastEventDate) {
				eventGroup.LastEventDate = &event.Datetime
			}
			if eventGroup.FirstEventDate == nil || event.Datetime.Before(*eventGroup.FirstEventDate) {
				eventGroup.FirstEventDate = &event.Datetime
			}
		}
		eventGroup.Events = events
		eventGroup.NumberOfEvents = counter
		eventGroup.MinNbOfOccurrence = endpoint.MinNbOfOccurrence
		eventList[endpoint.EventKey] = eventGroup
	}
	return
}

// MeetTriggerConditions checks if the currently stored events meet the conditions to trigger a trigger. If so, the
// needTrigger output is True and the eventList contains the events to put in the trigger
func (e *EventService) MeetTriggerConditions(ctx context.Context) (eventList map[string]models.EventList, needTrigger bool, err error) {

	if e.configService.GetConfig().Trigger.Type == models.TriggerTypeNone {
		fmt.Println("TriggerType set to None. No automatic evaluation")
		return nil, false, nil
	}

	// Get the list of event in the observation period
	eventList, err = e.GetEventsOverAPeriod(ctx, e.configService.GetConfig().Trigger.ObservationPeriod)
	if err != nil {
		return
	}

	// Evaluate against the configuration if all the events are here to trigger a new event.
	return eventList, e.checkTriggerConditions(eventList), nil
}

// checkTriggerConditions uses a list of events and validate against the configuration the requirement to trigger a
// trigger
func (e *EventService) checkTriggerConditions(eventList map[string]models.EventList) (needTrigger bool) {
	for _, endpoint := range e.configService.GetConfig().Endpoints {
		if _, ok := eventList[endpoint.EventKey]; !ok || eventList[endpoint.EventKey].NumberOfEvents == 0 {
			fmt.Printf("missing event entry for endpoint %s. Conditions are not met for a trigger\n", endpoint.EventKey)
			return false
		}

		if endpoint.MinNbOfOccurrence > eventList[endpoint.EventKey].NumberOfEvents {
			fmt.Printf("minimal number of event not satisfied for endpoint %s. Minimum is %d, got %d \n", endpoint.EventKey, endpoint.MinNbOfOccurrence, eventList[endpoint.EventKey].NumberOfEvents)
			return false
		}
	}
	return true
}

// MatchEndpoint checks if the current provided eventKeyValue meets one on the endpoints set in the configuration. If
// so, return True.
func (e *EventService) MatchEndpoint(eventKeyValue string) bool {
	for _, endpoint := range e.configService.GetConfig().Endpoints {
		if endpoint.EventKey == eventKeyValue {
			return true
		}
	}
	return false
}

// ResetEvents updates the events provided in the eventList to set the parameter AlreadyExported to True. Like that
// the event won't be retrieved during the future queries.
func (e *EventService) ResetEvents(ctx context.Context, eventList map[string]models.EventList) {

	fmt.Printf("set all the events has already exported to reset the context.\n")

	update := []firestore.Update{
		{
			Path:  "AlreadyExported",
			Value: true,
		},
	}

	for _, eventGroup := range eventList {
		for _, event := range eventGroup.Events {
			_, err := e.firestoreClient.Collection(e.configService.GetConfig().ServiceName).Doc(event.FirestoreDocumentID).Update(ctx, update)
			if err != nil {
				fmt.Printf("impossible to update the state of the documentID %s, with error %s", event.FirestoreDocumentID, err)
			}
			fmt.Printf("messageID %s set to already exported", event.FirestoreDocumentID)
		}
	}

}
