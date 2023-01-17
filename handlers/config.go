package handlers

import (
	"encoding/json"
	"eventsync/services"
	"net/http"
)

// ConfigHandler is the URL request handler for the configuration
type ConfigHandler struct {
	// ConfigService is the service to manage the configuration of the current instance
	ConfigService *services.ConfigService
}

// Config is the function to handle the config export request
func (c *ConfigHandler) Config(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(c.ConfigService.GetConfig())
}
