package main

import (
	"context"
	"eventsync/handlers"
	"eventsync/services"
	"log"
	"net/http"
	"os"
)

func main() {

	config := os.Getenv(services.ConfigEnvVar)

	configService, err := services.LoadConfig(config)
	if err != nil {
		log.Fatalf("impossible to load the configuration with error: %s\n", err)
	}

	err = configService.CheckConfig()
	if err != nil {
		log.Fatalf("invalid configuration with error %s\n", err)
	}

	ctx := context.Background()

	eventService, err := services.NewEventService(ctx, configService)
	if err != nil {
		log.Fatalf("impossible to create the event service with error %s\n", err)
	}

	triggerService, err := services.NewTriggerService(ctx, configService, eventService)
	if err != nil {
		log.Fatalf("impossible to create the trigger service with error %s\n", err)
	}

	configHandler := handlers.ConfigHandler{ConfigService: configService}
	eventHandler := handlers.EventHandler{EventService: eventService, ConfigService: configService, TriggerService: triggerService}
	resetHandler := handlers.ResetHandler{ConfigService: configService, EventService: eventService}
	triggerHandler := handlers.TriggerHandler{ConfigService: configService, TriggerService: triggerService, EventService: eventService}

	// To accept event, a dedicated endpoints is reserved to this.
	http.HandleFunc(services.EventPathPrefix, eventHandler.Event)
	http.HandleFunc("/config", configHandler.Config)
	http.HandleFunc("/trigger", triggerHandler.Trigger)
	http.HandleFunc("/reset", resetHandler.Reset)

	http.ListenAndServe(":8080", nil)

}
