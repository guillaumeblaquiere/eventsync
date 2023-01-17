package utils

import (
	"context"
	"fmt"
	"golang.org/x/oauth2/google"
	"os"
)

const projectIdKeyEnvVar = "projectID"

// GetProjectId extracts the project ID from the server metadata on Google Cloud (through credential) else try to finc
// the value in the projectIdKeyEnvVar environment variable
func GetProjectId() (projectID string) {
	//Test the GCP environment first
	ctx := context.Background()

	credentials, err := google.FindDefaultCredentials(ctx)
	if err != nil {
		fmt.Printf("impossible to get the default credentials with error %s\n", err)
		return
	}
	projectID = credentials.ProjectID

	//Check the Environment Variables
	if projectID == "" {
		projectID = os.Getenv(projectIdKeyEnvVar)
	}

	if projectID == "" {
		fmt.Println("WARNING: ProjectID not detected!")
	}

	return
}
