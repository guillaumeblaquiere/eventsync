package handlers

import (
	"eventsync/services"
	"eventsync/utils"
	"fmt"
	"net/http"
)

// TriggerHandler is the URL request handler for the configuration
type TriggerHandler struct {
	// ConfigService is the service to manage the configuration of the current instance
	ConfigService *services.ConfigService
	// TriggerService is the service to manage the event generation and formatting
	TriggerService *services.TriggerService
	// EventService is the service to manage, store and retrieve events
	EventService *services.EventService
}

// Trigger is the function to force the trigger by API request.
func (t *TriggerHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	utils.EnableCors(&w)

	eventList, err := t.EventService.GetEventsOverAPeriod(r.Context(), t.ConfigService.GetConfig().Trigger.ObservationPeriod)
	if err != nil {
		fmt.Printf("impossible to retrive the list of events with error %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "impossible to retrive the list of events with error %s\n", err)
		return
	}

	err = t.TriggerService.TriggerEvent(r.Context(), eventList)
	if err != nil {
		fmt.Printf("impossible to trigger the events with error %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "impossible to trigger the events with error %s\n", err)
		return
	}

	fmt.Fprintf(w, "trigger correctly performed.")
}
