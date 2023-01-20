package models

import (
	"time"
)

// EventList is the list of events of each eventKey
type EventList struct {
	// FirstEventDate is the oldest event logged in the Trigger's observation period.
	// This event might be not included in the Events array, according to the endpoint configuration
	FirstEventDate *time.Time `json:"firstEventDate,omitempty"`
	// LastEventDate is the youngest event logged in the Trigger's observation period
	// This event might be not included in the Events array, according to the endpoint configuration
	LastEventDate *time.Time `json:"lastEventDate,omitempty"`
	// NumberOfEvents is the number of logged events over the Trigger's observation period
	// This number of events might be match the Events array size, according to the endpoint configuration.
	NumberOfEvents int `json:"numberOfEvents"`
	// MinNbOfOccurrence is the value set in the configuration to consider the endpoint valid
	MinNbOfOccurrence int `json:"minNbOfOccurrence"`
	// EventToSend is the value set in the configuration to define the event to add in the generated event sync message
	EventToSend EventToSendType `json:"eventToSend"`
	// Events is the list of Event over the Trigger's observation period and that match the endpoint configuration
	Events []Event `json:"events,omitempty"`
}

// EventGenerated is the structure of message published in the Target configuration
type EventGenerated struct {
	// EventID is the unique identifier of the ID based on a MD5 hash of all the messages in Events
	EventID string `json:"eventID"`
	// Date is the timestamp of the generated event
	Date time.Time `json:"date"`
	// ServiceName is the name of the current service name configuration
	ServiceName string `json:"serviceName"`
	// TriggerTpe is the type of trigger (TriggerTypeWindow or TriggerTypeNone)
	TriggerTpe triggerType `json:"triggerType"`
	// Events is the list of events of each eventKey.
	Events map[string]EventList `json:"events"` //key is the eventKey
}
