package models

import (
	"time"
)

// Event is the persistent representation of the HTTP event received
type Event struct {
	// AlreadyExported is the status of the event. Not exported in JSON
	AlreadyExported bool `json:"-"`
	// FirestoreDocumentID is the documentID of the Firestore document. It's a transient value, never stored or
	// exported. Only for internal processing when the event has to be reset.
	FirestoreDocumentID string `json:"-" firestore:"-"`
	// Datetime is the date of the event reception by the application
	Datetime time.Time `json:"datetime"`
	// EventKey is the endpoint on which the event has been sent, represented by the eventKey
	EventKey string `json:"eventKey"`
	// Headers represents the headers of the event HTTP request
	Headers map[string][]string `json:"headers,omitempty"`
	// QueryParams represents the query parameters of the event HTTP request
	QueryParams map[string][]string `json:"queryParams,omitempty"`
	// Content is the body content of the event HTTP request
	Content interface{} `json:"content,omitempty"`
	// Method is the HTTP method of the event HTTP request
	Method HttpMethodType `json:"httpMethod"`
}
