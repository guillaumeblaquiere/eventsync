package handlers

import (
	"context"
	"errors"
	"eventsync/services"
	"fmt"
	"net/http"
)

// EventHandler is the URL request handler for the events' acquisition
type EventHandler struct {
	// EventService is the service to manage, store and retrieve events
	EventService *services.EventService
	// ConfigService is the service to manage the configuration of the current instance
	ConfigService *services.ConfigService
	// TriggerService is the service to manage the event generation and formatting
	TriggerService *services.TriggerService
}

// Event is the function to handle the event acquisition request
func (e *EventHandler) Event(w http.ResponseWriter, r *http.Request) {

	// extract the eventKey
	eventKeyValue := services.ExtractEventKey(r.URL.Path)

	if eventKeyValue == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "missing or incorrect eventKey in the path")
		return
	}

	// check if accepted endpoint
	if e.EventService.MatchEndpoint(eventKeyValue) {

		//If the query param match the configuration, store the formatted event
		event, err := services.FormatEvent(eventKeyValue, r.URL.Query(), r.Header, r.Body)
		if err != nil {
			fmt.Fprintf(w, "incorrect event format")
			return
		}

		err = e.EventService.StoreEvent(r.Context(), event)
		if err != nil {
			fmt.Fprintf(w, "impossible to store the event %v in the collection %s, with error %s\n", event, e.ConfigService.GetConfig().ServiceName, err)
			return
		}
		fmt.Fprintf(w, "event correct stored to Firestore collection %s\n", e.ConfigService.GetConfig().ServiceName)

		//FIXME could perform this task async with Always on option on Cloud Run
		err = e.postProcessEvent(r.Context())
		if err != nil {
			fmt.Fprintf(w, err.Error())
		}

		return

	}
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, "incorrect value for eventKey the path")

}

// postProcessEvent performs processing after the correct storage of the event, like checking if an event sync has
// to be generated
func (e *EventHandler) postProcessEvent(ctx context.Context) (err error) {
	eventList, needTrigger, err := e.EventService.MeetTriggerConditions(ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("impossible to check the trigger conditions with error: %s\n", err))
	}

	if needTrigger {
		err = e.TriggerService.TriggerEvent(ctx, eventList)
		if err != nil {
			return errors.New(fmt.Sprintf("no trigger done after the event storage\n"))
		}
	}
	return
}
