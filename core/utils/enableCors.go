package utils

import (
	"net/http"
	"os"
)

// Deactivate CORS only for browser and demo purpose. The env var 'disableCORS' must be set to anything to activate the
// feature
func EnableCors(w *http.ResponseWriter) {
	disableCors := os.Getenv("DISABLE_CORS")
	if disableCors != "" {
		(*w).Header().Set("Access-Control-Allow-Origin", "*")
	}
}
