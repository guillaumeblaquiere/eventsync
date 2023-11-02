package handlers

import (
	"eventsync/services"
	"eventsync/utils"
	"fmt"
	"net/http"
)

// ResetHandler is the URL request handler for the configuration
type ResetHandler struct {
	// ConfigService is the service to manage the configuration of the current instance
	ConfigService *services.ConfigService
	// EventService is the service to manage, store and retrieve events
	EventService *services.EventService
}

// Reset is the function to handle the reset request, to cancel all the previous event over the configured observation
// period
func (rh *ResetHandler) Reset(w http.ResponseWriter, r *http.Request) {
	utils.EnableCors(&w)

	events, err := rh.EventService.GetEventsOverAPeriod(r.Context(), rh.ConfigService.GetConfig().Trigger.ObservationPeriod)
	if err != nil {
		fmt.Printf("impossible to get the events to reset with error %s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "impossible to get the events to reset with error %s\n", err)
		return
	}

	rh.EventService.ResetEvents(r.Context(), events)

	fmt.Fprintf(w, "the events have been correctly reset.\n")
}
