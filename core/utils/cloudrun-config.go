package utils

import (
	"cloud.google.com/go/compute/metadata"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/oauth2/google"
	"io"
	"os"
	"strings"
)

const kServiceEnvVar = "K_SERVICE"

// getCloudRunProjectNumberAndRegion extracts from the Cloud Run metadata the project Number and the Google Cloud
// region on which Cloud Run runs. If the metadata server is missing (not running on Cloud Run) an error is raised.
func getCloudRunProjectNumberAndRegion() (projectNumber string, region string, err error) {
	resp, err := metadata.Get("/instance/region")
	if err != nil {
		fmt.Printf("impossible to get the metadata values with error %s\n", err)
		return
	}
	// response pattern is projects/<projectNumber>/regions/<region>
	r := strings.Split(resp, "/")
	projectNumber = r[1]
	region = r[3]
	return
}

// getCloudRunServiceName extracts the Cloud Run service name from the runtime environment. If it does not run on
// Cloud Run (or KNative compliant environment) the return is empty string ""
func getCloudRunServiceName() string {
	return os.Getenv(kServiceEnvVar)
}

// getCloudRunJsonConfig extracts the current runtime configuration of Cloud Run revision. The full JSON is returned.
// An error is raised, especially if it does not run on Cloud Run.
func getCloudRunJsonConfig() (data []byte, err error) {
	service := getCloudRunServiceName()
	if service == "" {
		err = errors.New(fmt.Sprintf("no %q env var detected. Should not run on Cloud Run. Impossible to get the envitonment config", kServiceEnvVar))
		return
	}

	projectNumber, region, err := getCloudRunProjectNumberAndRegion()

	ctx := context.Background()
	client, err := google.DefaultClient(ctx)

	cloudRunApi := fmt.Sprintf("https://%s-run.googleapis.com/apis/serving.knative.dev/v1/namespaces/%s/services/%s", region, projectNumber, service)
	resp, err := client.Get(cloudRunApi)

	if err != nil {
		fmt.Printf("impossible to get the Cloud Run service configuration by API call with error %s\n", err)
		return
	}
	defer resp.Body.Close()

	data, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("impossible to read the Cloud Run service configuration with error %s\n", err)
		return
	}
	return
}

// IsCloudRunCPUThrottled checks the current Cloud Run configuration and return is Cloud Run CPU Always ON feature
// is activated (false) or not (true). If the app does not run on Cloud Run, true (CPU Throttled) is returned by default
func IsCloudRunCPUThrottled() (throttled bool) {
	config, err := getCloudRunJsonConfig()
	if err != nil {
		fmt.Printf("impossible to get the Cloud Run config. Set the throttled to TRUE by default (thread safe solution)")
		return true
	}
	cloudRunResp := &cloudRunAPIAnnotationThrottlingOnly{}
	err = json.Unmarshal(config, cloudRunResp)
	if err != nil {
		fmt.Printf("impossible to get the Cloud Run Cpu Throttling config. Set the throttled to TRUE by default (thread safe solution)")
		return true
	}
	data := cloudRunResp.Spec.Template.Metadata.Annotations.RunGoogleapisComCPUThrottling

	return data == "" || strings.ToLower(data) == "true"
}

// cloudRunAPIAnnotationThrottlingOnly is the minimal struct type to get only the interesting part in the Cloud Run
// JSON configuration
type cloudRunAPIAnnotationThrottlingOnly struct {
	Spec struct {
		Template struct {
			Metadata struct {
				Name        string `json:"name"`
				Annotations struct {
					RunGoogleapisComCPUThrottling string `json:"run.googleapis.com/cpu-throttling"`
				} `json:"annotations"`
			} `json:"metadata"`
		} `json:"template"`
	} `json:"spec"`
}
