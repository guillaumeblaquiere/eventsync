package main

import (
	"cloud.google.com/go/pubsub"
	"context"
	"fmt"
	"golang.org/x/net/websocket"
	"golang.org/x/oauth2/google"
	"log"
	"net/http"
	"os"
)

// PubSub Subscription name must be set in env var
const subscriptionEnvVar = "SUBSCRIPTION"

// ProjectID in case of user credential usage (for local development/run)
const projectIDEnvVar = "PROJECT_ID"

// Singleton PubSub client
var client *pubsub.Client

// Subscription name (extracted from env var)
var subscriptionName string

func main() {

	subscriptionName = os.Getenv(subscriptionEnvVar)
	if subscriptionName == "" {
		log.Fatalln("a subscription name must be set in argument")
	}

	ctx := context.Background()
	var err error

	// Get the project ID from credential or from Env Var
	projectID := getProjectId(ctx)

	// create PubSub client and subscription
	client, err = pubsub.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Check if the subscription exist
	ok, err := client.Subscription(subscriptionName).Exists(ctx)
	if err != nil {
		log.Fatalf("Failed to check subscription existence: %v", err)
	}
	if !ok {
		log.Fatalf("pubsub subscription %s must exist in Pull mode", subscriptionName)
	} else {
		// Check if the PubSub subscription is in pull mode
		sub, err := client.Subscription(subscriptionName).Config(ctx)
		if err != nil {
			log.Fatalf("Failed to get subscription config: %v", err) // TODO
		}
		if sub.PushConfig.Endpoint != "" {
			log.Fatalf("pubsub subscription %s must be in Pull mode", subscriptionName)
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/", websocket.Handler(ws))

	http.ListenAndServe(":8080", mux)
}

func ws(ws *websocket.Conn) {
	fmt.Println("New connection")

	// Get the subscription
	subscription := client.Subscription(subscriptionName)

	// Send welcome message with the subscription name
	err := websocket.Message.Send(ws, fmt.Sprintf("{\"subscription\" : \"%s\"}", subscription.String()))
	if err != nil {
		fmt.Println("Can't send")
		return
	}

	// Create a cancelable context to handle the websocket client-side disconnection
	ctx, cancel := context.WithCancel(context.Background())

	// Receive message from the client. Anything means goodbye
	go func() {
		var msg string
		err := websocket.Message.Receive(ws, &msg)
		fmt.Println("Received from client: " + msg)
		if err != nil {
			fmt.Println("Can't receive")
		}
		// Cancel the context
		cancel()
		// Close the websocket
		ws.Close()
		return
	}()

	// Receive messages from the pubsub subscription and until context cancellation.
	subscription.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		// Get the message data, assume it is in JSON format
		data := string(msg.Data)

		// Forward all message data to the websocket client
		err := websocket.Message.Send(ws, data)
		if err != nil {
			fmt.Println("Can't send")
			return
		}
		//keep local logs for debugs
		fmt.Printf(fmt.Sprintf("sent: %s\n", data))
		msg.Ack()
	})

}

// Get the project ID from credential or from Env Var (for local environment)
func getProjectId(ctx context.Context) string {
	projectID := os.Getenv(projectIDEnvVar)
	if projectID == "" {
		credentials, err := google.FindDefaultCredentials(ctx)
		if err != nil {
			log.Fatalf("Failed to create client: %v", err)
		}
		if credentials.ProjectID == "" {
			log.Fatalln("impossible to find the default project")
		}
		projectID = credentials.ProjectID
	}
	return projectID
}
