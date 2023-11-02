package services

import (
	"cloud.google.com/go/pubsub"
	"context"
	"crypto/md5"
	"encoding/json"
	"eventsync/models"
	"fmt"
	"strings"
	"time"
)

// TriggerService is in charge to submit the events when a trigger need to be performed.
type TriggerService struct {
	configService *ConfigService
	eventService  *EventService
	pubsubTopic   *pubsub.Topic
}

// NewTriggerService creates a TriggerService instance. The context is required to create a PubSub client and a Topic
// object to be able to send PubSub message when required.
func NewTriggerService(ctx context.Context, configService *ConfigService, eventService *EventService) (triggerService *TriggerService, err error) {
	triggerService = &TriggerService{
		configService: configService,
		eventService:  eventService,
	}

	//TODO create the firestoreClient only if the PubSub trigger type is used

	//Topic Split size must be 4. The check has been performed during the load config
	topicSplit := strings.Split(configService.GetConfig().TargetPubSub.Topic, "/")

	client, err := pubsub.NewClient(ctx, topicSplit[1])
	if err != nil {
		fmt.Printf("pubsub new firestoreClient error:%s\n", err)
		return nil, err
	}
	// Only the topic is kept in memory. The PubSub client is not required
	triggerService.pubsubTopic = client.Topic(topicSplit[3])

	return
}

// TriggerEvent generates a models.EventGenerated object based on the events and send it through the configured
// trigger channels (only PubSub for now)
func (t *TriggerService) TriggerEvent(ctx context.Context, events map[string][]models.Event) (err error) {

	eventGenerated := t.createEventGenerated(events)

	// Send it according to the target configuration
	if t.configService.GetConfig().TargetPubSub != nil {
		err = t.triggerPubSub(ctx, &eventGenerated)
		if err != nil {
			return err
		}
	}

	fmt.Printf("keep the events after the trigger set to %v\n", t.configService.GetConfig().Trigger.KeepEventAfterTrigger)
	//Cleanup the context
	if !t.configService.GetConfig().Trigger.KeepEventAfterTrigger {
		t.eventService.ResetEvents(ctx, events)
	} else {
		fmt.Printf("no clean-up to do\n")
	}

	return
}

// triggerPubSub effectively format the eventGenerated parameter into a PubSub message and submit it to the topic.
// The serviceName is added as attribute
func (t *TriggerService) triggerPubSub(ctx context.Context, eventGenerated *models.EventGenerated) (err error) {

	data, err := json.Marshal(eventGenerated)
	if err != nil {
		fmt.Printf("impossible to generate the PubSub message with error:%s\n", err)
		return
	}

	fmt.Printf("content to send to PubSub: %s\n", string(data))

	message := &pubsub.Message{
		Data: data,
		Attributes: map[string]string{
			"serviceName": t.configService.GetConfig().ServiceName,
		},
	}

	result := t.pubsubTopic.Publish(ctx, message)
	if _, err = result.Get(ctx); err != nil {
		fmt.Printf("impossible to publish the message %s with error:%s\n", string(data), err)
	} else {
		fmt.Printf("event sent to topic %s\n", t.configService.GetConfig().TargetPubSub.Topic)
	}

	return
}

// createEventGenerated produces an eventGenerated structure based on the events in entry and the configuration
// of the endpoints. Some metrics are extracted such as firstEventDate, LastEventDate, number of events.
// Other configuration option are duplicated to help the consumer of the message to understand the context.
// A unique EventID is generated based on the FirestoreIDs of the events included in the eventGenerated message. That
// event help the consumer to deduplicate the messages, if any.
func (t *TriggerService) createEventGenerated(events map[string][]models.Event) (eventGenerated models.EventGenerated) {

	eventGenerated = models.EventGenerated{
		Date:        time.Now(),
		Events:      make(map[string]*models.EventList, len(t.configService.GetConfig().Endpoints)),
		ServiceName: t.configService.GetConfig().ServiceName,
		TriggerTpe:  t.configService.GetConfig().Trigger.Type,
	}

	eventIds := ""
	for _, endpoint := range t.configService.GetConfig().Endpoints {
		eventList := &models.EventList{
			MinNbOfOccurrence: endpoint.MinNbOfOccurrence,
			EventToSend:       endpoint.EventToSend,
		}

		var firstEvent, lastEvent models.Event
		counter := 0
		for _, event := range events[endpoint.EventKey] {

			// If all the event are kept, aggregate all the IDs
			if endpoint.EventToSend == models.EventToSendTypeAll {
				eventIds += event.FirestoreDocumentID
			}

			if eventList.LastEventDate == nil || event.Datetime.After(*eventList.LastEventDate) {
				d := event.Datetime
				eventList.LastEventDate = &d
				lastEvent = event
			}
			if eventList.FirstEventDate == nil || event.Datetime.Before(*eventList.FirstEventDate) {
				d := event.Datetime
				eventList.FirstEventDate = &d
				firstEvent = event
			}
			counter += 1

		}

		// Keep only the event to send if the counter is > 0
		if counter > 0 {
			switch endpoint.EventToSend {
			case models.EventToSendTypeAll:
				eventList.Events = events[endpoint.EventKey]
			case models.EventToSendTypeFirst:
				eventList.Events = []models.Event{firstEvent}
				eventIds += firstEvent.FirestoreDocumentID
			case models.EventToSendTypeLast:
				eventList.Events = []models.Event{lastEvent}
				eventIds += lastEvent.FirestoreDocumentID
			case models.EventToSendTypeBoundaries:
				// If there is only one element, add only one, else add the boudaries
				if firstEvent.FirestoreDocumentID == lastEvent.FirestoreDocumentID {
					eventList.Events = []models.Event{firstEvent}
					eventIds += firstEvent.FirestoreDocumentID
				} else {
					eventList.Events = []models.Event{firstEvent, lastEvent}
					eventIds += firstEvent.FirestoreDocumentID + lastEvent.FirestoreDocumentID
				}
			}
		}
		eventList.NumberOfEvents = counter
		eventGenerated.Events[endpoint.EventKey] = eventList
	}

	// EventID is generated with the MD5 hash of the string composed of event's firestoreID contains in event sync
	// message generated
	fmt.Println(eventIds)
	eventGenerated.EventID = fmt.Sprintf("%x", md5.Sum([]byte(eventIds)))

	return
}
