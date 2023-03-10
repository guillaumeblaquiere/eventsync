package services

import (
	"cloud.google.com/go/firestore"
	apiAdmin "cloud.google.com/go/firestore/apiv1/admin"
	"cloud.google.com/go/firestore/apiv1/admin/adminpb"
	"context"
	"errors"
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
func FormatEvent(eventKey string, queryParam map[string][]string, headers map[string][]string, body io.ReadCloser, method string) (event models.Event, err error) {
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
	event.Method = models.HttpMethodType(method)
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
// alreadyExported event are taken into account. The events output groups the events per eventKeys.
func (e *EventService) GetEventsOverAPeriod(ctx context.Context, observationPeriod int64) (events map[string][]models.Event, err error) {

	//Define globally the time of reference
	targetDatetime := time.Now().Add(-time.Duration(observationPeriod) * time.Second)

	// The base query select the events in the observation period and not already exported
	query := e.firestoreClient.Collection(e.configService.GetConfig().ServiceName).Where("Datetime", ">", targetDatetime).Where("AlreadyExported", "==", false)

	events = make(map[string][]models.Event, len(e.configService.GetConfig().Endpoints))

	//Perform Query for all endpoints in th config
	for _, endpoint := range e.configService.GetConfig().Endpoints {
		// Specialize the query to request per eventKey
		iter := query.Where("EventKey", "==", endpoint.EventKey).Documents(ctx)

		// Initialize the  list of rawEvents of that eventKey
		rawEvents := make([]models.Event, 0)
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
			// Count the number of rawEvents in the observation period
			event := &models.Event{}
			err = doc.DataTo(&event)
			if err != nil {
				fmt.Printf("error during the document conversion with error: %s\n", err)
				return
			}
			// Keep the documentID for later use
			event.FirestoreDocumentID = doc.Ref.ID
			rawEvents = append(rawEvents, *event)
		}
		events[endpoint.EventKey] = rawEvents
	}
	return
}

// MeetTriggerConditions checks if the currently stored events meet the conditions to trigger a trigger. If so, the
// needTrigger output is True and the events contains the events to put in the trigger
func (e *EventService) MeetTriggerConditions(ctx context.Context) (events map[string][]models.Event, needTrigger bool, err error) {

	if e.configService.GetConfig().Trigger.Type == models.TriggerTypeNone {
		fmt.Println("TriggerType set to None. No automatic evaluation")
		return nil, false, nil
	}

	// Get the list of event in the observation period
	events, err = e.GetEventsOverAPeriod(ctx, e.configService.GetConfig().Trigger.ObservationPeriod)
	if err != nil {
		return
	}

	// Evaluate against the configuration if all the events are here to trigger a new event.
	return events, e.checkTriggerConditions(events), nil
}

// checkTriggerConditions uses a list of events and validate against the configuration the requirement to trigger a
// trigger
func (e *EventService) checkTriggerConditions(events map[string][]models.Event) (needTrigger bool) {

	for _, endpoint := range e.configService.GetConfig().Endpoints {
		numberOfEvents := len(events[endpoint.EventKey])
		if _, ok := events[endpoint.EventKey]; !ok || numberOfEvents == 0 {
			fmt.Printf("missing event entry for endpoint %s. Conditions are not met for a trigger\n", endpoint.EventKey)
			return false
		}

		if endpoint.MinNbOfOccurrence > numberOfEvents {
			fmt.Printf("minimal number of event not satisfied for endpoint %s. Minimum is %d, got %d \n", endpoint.EventKey, endpoint.MinNbOfOccurrence, numberOfEvents)
			return false
		}
	}
	return true
}

// MatchEndpoint checks if the current provided eventKeyValue meets one on the endpoints set in the configuration. If
// so, return True.
func (e *EventService) MatchEndpoint(eventKeyValue string, method string) (err error) {
	for _, endpoint := range e.configService.GetConfig().Endpoints {
		if endpoint.EventKey == eventKeyValue {
			if contains(endpoint.AcceptedHttpMethods, models.HttpMethodType(method)) {
				return nil
			} else {
				return errors.New(fmt.Sprintf("invalid method %q for endpoint %q\n", method, eventKeyValue))
			}
		}
	}
	return errors.New(fmt.Sprintf("invalid endpoint %q\n", eventKeyValue))
}

func contains(s []models.HttpMethodType, str models.HttpMethodType) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

// ResetEvents updates the events provided in the events to set the parameter AlreadyExported to True. Like that
// the event won't be retrieved during the future queries.
func (e *EventService) ResetEvents(ctx context.Context, events map[string][]models.Event) {

	fmt.Printf("set all the events has already exported to reset the context.\n")

	update := []firestore.Update{
		{
			Path:  "AlreadyExported",
			Value: true,
		},
	}

	for _, eventGroup := range events {
		for _, event := range eventGroup {
			_, err := e.firestoreClient.Collection(e.configService.GetConfig().ServiceName).Doc(event.FirestoreDocumentID).Update(ctx, update)
			if err != nil {
				fmt.Printf("impossible to update the state of the documentID %s, with error %s", event.FirestoreDocumentID, err)
			}
			fmt.Printf("messageID %s set to already exported", event.FirestoreDocumentID)
		}
	}

}
