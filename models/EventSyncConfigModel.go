package models

/*------------------*/

// Endpoint is the definition of event acquisition path
type Endpoint struct {
	// EventKey is the URL path suffix to add to submit that type of events to the app
	EventKey string `json:"eventKey"`
	//AcceptedHttpVerbs
	//Filter string `json:"filter"`
	//KeepAllValues bool `json:"keepAllValues"`

	// MinNbOfOccurrence is the minimal number of events to consider the endpoint valid for an automatic trigger.
	// The value must be > 0. If it is omitted or set to 0, it is set to 1 by default.
	MinNbOfOccurrence int `json:"minNbOfOccurrence"`
	// TODO is optional?
}

/*------------------*/
// triggerType is the different type of possible Trigger
type triggerType string

const (
	// TriggerTypeWindow sets an automatic event sync configuration when all the endpoints are compliant over the
	// ObservationPeriod.
	TriggerTypeWindow triggerType = "window"
	// TriggerTypeNone discards automatic event trigger. Only API calls can trigger the events, even if all the
	// automatic event conditions aren't met.
	TriggerTypeNone = "none"
)

// Trigger is the configuration to meet to send a new event
type Trigger struct {
	// The Type of the trigger. Must be a "window" or "none"
	Type triggerType `json:"type"`
	// ObservationPeriod over which the events are get. Must be > 0
	ObservationPeriod int64 `json:"observationPeriod"`
	// KeepEventAfterTrigger defines if an event can be taken into account for a subsequent sync event after being
	// exported.
	KeepEventAfterTrigger bool `json:"keepEventAfterTrigger"`
}

/*------------------*/

// TargetPubSub is the pubsub configuration to publish an event.
type TargetPubSub struct {
	// Topic is the fully qualified name of the PubSub topic to publish the event sync message. The format must be
	// `projects/<ProjectID>/topics/<TopicName>`
	Topic string `json:"topic"`
}

/*------------------*/

// EventSyncConfig is the configuration representation of the current service
type EventSyncConfig struct {
	// ServiceName is the name of the service, also use to create the Firestore collection
	ServiceName string `json:"serviceName"`
	// Endpoints is the list of different event required to sync before triggering a new one
	Endpoints []Endpoint `json:"endpoints"`
	// Trigger is the conditions to meet for triggering a new event
	Trigger *Trigger `json:"trigger"`
	// TargetPubSub is the PubSub configuration to publish the new event in.
	TargetPubSub *TargetPubSub `json:"targetPubSub"`
}
