package services

import (
	"encoding/json"
	"errors"
	"eventsync/models"
	"fmt"
	"strings"
)

// ConfigService in the configuration of EventSync instance. It contains the detail of the loaded configuration
type ConfigService struct {
	eventSyncConfig *models.EventSyncConfig
}

const ConfigEnvVar = "CONFIG"

// LoadConfig creates a ConfigService based on the JSON config in parameter.
func LoadConfig(config string) (conf *ConfigService, err error) {

	conf = &ConfigService{
		eventSyncConfig: &models.EventSyncConfig{},
	}
	err = json.Unmarshal([]byte(config), conf.eventSyncConfig)
	return
}

// CheckConfig verifies if the provided JSON configuration is operationally correct. A description of the configuration
// or the list of errors is displayed in the logs.
func (c *ConfigService) CheckConfig() (err error) {

	logOK := "The configuration is correct. You have set:\n"
	logKO := ""

	logKO, logOK = c.checkConfigRootValues(logKO, logOK)

	logKO, logOK = c.checkConfigTargetPubSub(logKO, logOK)

	logKO, logOK = c.checkConfigEndpoints(logKO, logOK)

	logKO, logOK = c.checkConfigTrigger(logKO, logOK)

	if logKO != "" {
		return errors.New("The configuration contains one or several blocking errors. Here the list:\n" + logKO)
	}
	fmt.Println(logOK)
	return
}

// checkConfigTrigger checks if the provided trigger configuration is correct and return the corresponding log strings
func (c *ConfigService) checkConfigTrigger(logKO string, logOK string) (string, string) {

	// A trigger must exist
	if c.eventSyncConfig.Trigger == nil {
		logKO += fmt.Sprintf("The trigger conditions and type must be set\n")
	} else {
		logOK += fmt.Sprintf("The trigger conditions are:\n")

		// ObservationPeriod must be above 0
		if c.eventSyncConfig.Trigger.ObservationPeriod <= 0 {
			logKO += fmt.Sprintf("The ObservationPeriod of the trigger must be > 0\n")
		} else {
			logOK += fmt.Sprintf("  - The ObservationPeriod is set to %d seconds\n", c.eventSyncConfig.Trigger.ObservationPeriod)
		}

		// The trigger type must be this one accepted
		if c.eventSyncConfig.Trigger.Type != models.TriggerTypeNone &&
			c.eventSyncConfig.Trigger.Type != models.TriggerTypeWindow {
			logKO += fmt.Sprintf("The type of the trigger must be %q (based on the observation period and the list of endpoints) or %q (only manual/by API trigger)\n", models.TriggerTypeWindow, models.TriggerTypeNone)
		} else {
			logOK += fmt.Sprintf("  - The type of the trigger is %q\n", c.eventSyncConfig.Trigger.Type)
		}

		// Only for nicer logs
		if c.eventSyncConfig.Trigger.KeepEventAfterTrigger {
			logOK += fmt.Sprintf("  - The events are kept (and could be resent or count for a subsequent trigger)\n")
		} else {
			logOK += fmt.Sprintf("  - The events are exported only once\n")
		}

	}
	return logKO, logOK
}

// checkConfigEndpoints checks if the provided endpoints configuration is correct and return the corresponding log
// strings
func (c *ConfigService) checkConfigEndpoints(logKO string, logOK string) (string, string) {

	// At least 2 endpoints must be defined
	if c.eventSyncConfig.Endpoints == nil || len(c.eventSyncConfig.Endpoints) < 2 {
		logKO += fmt.Sprintf("The endpoints definition must contains at least 2 entries\n")
	} else {
		logOK += fmt.Sprintf("The defined endpoints are:\n")
		for i, endpoint := range c.eventSyncConfig.Endpoints {
			logOK += fmt.Sprintf("  %d. The eventKey is %q. The path to reach for using it is %s%s\n", i+1, endpoint.EventKey, EventPathPrefix, endpoint.EventKey)

			if endpoint.EventKey == "" {
				logKO += fmt.Sprintf("The endpoint eventKey must not be empty at index %d\n", i)
			}

			// Check the endpoint eventKey unicity
			for j := 0; j < i; j++ {
				if endpoint.EventKey == c.eventSyncConfig.Endpoints[j].EventKey {
					logKO += fmt.Sprintf("The endpoints eventKeys must be unique. the eventKey %q is duplicated at index %d and %q\n", endpoint.EventKey, j, i)
				}
			}

			if endpoint.MinNbOfOccurrence < 0 {
				logKO += fmt.Sprintf("The minimal number of required event must be strictly positive for tne endpoint eventKey %q\n", endpoint.EventKey)
			}
			if endpoint.MinNbOfOccurrence == 0 {
				// Set one by default
				endpoint.MinNbOfOccurrence = 1
			}
			// Check the min instance
			logOK += fmt.Sprintf("    The minimal number of required event is set to %d\n", endpoint.MinNbOfOccurrence)

		}
	}
	return logKO, logOK
}

// checkConfigTargetPubSub checks if the provided targetPubSub configuration is correct and return the corresponding
// log strings
func (c *ConfigService) checkConfigTargetPubSub(logKO string, logOK string) (string, string) {

	// The PubSUb target must exist
	if c.eventSyncConfig.TargetPubSub == nil {
		logKO += fmt.Sprintf("The targetPubSub definition must be set\n")
	} else {
		logOK += fmt.Sprintf("The triggered event will be sent to PubSub:\n")

		// The topic format is the fully qualified name projects/<ProjectID>/topics/<TopicName>
		topicSplit := strings.Split(c.eventSyncConfig.TargetPubSub.Topic, "/")
		if len(topicSplit) != 4 {
			logKO += fmt.Sprintf("The topic format of must be \"projects/<ProjectID>/topics/<TopicName>\", here %q", c.eventSyncConfig.TargetPubSub.Topic)
		} else {
			logOK += fmt.Sprintf("  - The project of the topic is %q\n", topicSplit[1])
			logOK += fmt.Sprintf("  - The topic name is %q\n", topicSplit[3])
		}
	}
	return logKO, logOK
}

// checkConfigRootValues checks if the provided root configuration is correct and return the corresponding log strings
func (c *ConfigService) checkConfigRootValues(logKO string, logOK string) (string, string) {
	if c.eventSyncConfig.ServiceName == "" {
		logKO += fmt.Sprintf("The ServiceName must be set\n")
	} else {
		logOK += fmt.Sprintf("The service name set is %q, it will be use as collection in Firestore.\n", c.eventSyncConfig.ServiceName)
	}
	return logKO, logOK
}

// GetConfig returns the stored configuration of the service.
func (c *ConfigService) GetConfig() (eventSyncConfig *models.EventSyncConfig) {
	return c.eventSyncConfig
}
