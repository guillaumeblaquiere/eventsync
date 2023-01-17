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

// TriggerEvent generates a models.EventGenerated object based on the eventList and send it through the configured
// trigger channels (only PubSub for now)
func (t *TriggerService) TriggerEvent(ctx context.Context, eventList map[string]models.EventList) (err error) {

	// Format the output model of the event
	eventGenerated := &models.EventGenerated{
		EventID:     generateEventID(eventList),
		Date:        time.Now(),
		ServiceName: t.configService.GetConfig().ServiceName,
		TriggerTpe:  t.configService.GetConfig().Trigger.Type,
		Events:      eventList,
	}

	// Send it according to the target configuration
	if t.configService.GetConfig().TargetPubSub != nil {
		err = t.triggerPubSub(ctx, eventGenerated)
	}

	fmt.Printf("keep the events after the trigger set to %v\n", t.configService.GetConfig().Trigger.KeepEventAfterTrigger)
	//Cleanup the context
	if !t.configService.GetConfig().Trigger.KeepEventAfterTrigger {
		t.eventService.ResetEvents(ctx, eventList)
	} else {
		fmt.Printf("no clean-up to do\n")
	}

	return
}

func generateEventID(eventList map[string]models.EventList) (eventId string) {

	// EventIDs is the concatenation of all the Firebase IDs of the events. A hash of that string will be the EventID
	eventIds := ""
	for _, eventGroup := range eventList {
		for _, event := range eventGroup.Events {
			eventIds += event.FirestoreDocumentID
		}
	}
	eventId = fmt.Sprintf("%x", md5.Sum([]byte(eventIds)))
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
