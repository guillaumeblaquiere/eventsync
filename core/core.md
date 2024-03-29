# Overview

EventSync offers a synchronization between different event sources and, when all the conditions are met, a new event is
triggered. It has been designed and tested on Cloud Run. _But it can be hosted on other Google Cloud runtime environment 
with no or a few updates_

It's especially interesting to synchronize discontinuous event sources, from different projects or providers.

*This [article](https://medium.com/google-cloud/eventsync-the-event-driven-management-missing-piece-baeb4fcb9315) 
presents that product and the problems it tackles*

# General architecture

According to a configuration (see below), a number of endpoints are exposed. The event sources must reach the declared 
endpoints to allow EventSync to track the event and detect if the conditions are met to trigger a new event sync.

The sources can be a Cloud Scheduler HTTP call, a PubSub push subscription, an Eventarc source, a Workflow step or
whatever that can perform a simple HTTP call.

The security is enforced by Cloud Run service. **_Be sure that your event sources have the correct permission to invoke 
the deployed Cloud Run service._**

This application uses Firestore to persist the events and to check when it's the right time to trigger a new event sync.
To optimize the query, the app automatically creates the correct composite index in Firestore. ***Be careful, at the
start, it could take a few minutes before the end of the index creation and for having a fully operational solution***

There are 2 types of triggers:
* `Window`: this mode validates each endpoint and, if the conditions are met over the observation period a new event 
sync is generated and sent to the target. The check is performed after each event received on an endpoint. 
* `None`: only "manual" (by API call). In that case all the events stored over an observation period are retrieved and
sent in the new event sync message. ***This case is interesting to get all the event occurs over a period of time,
even if all the event on the different endpoints have not been received***

The generated event sync message contains the details of the eligible HTTP events: header, query param, body.

# Known limitations

The solution is not designed to handle events with a high throughput (more than 1 event per 500ms). You could have 
duplicated event sync messages in that case.

The `EventID` in the event sync message is the MD5 hash of all the events (_the FirestoreID in fact_) contained in the
event sync message. You can perform a deduplication on the consumer side if you want to avoid duplicates.

The target is based on PubSub. The max message size of PubSub is 10Mb. Therefore, the sum of all events included in the
event sync message generated must not be bigger than 10Mb.

# How to use

You can use the public image of the project: **gcr.io/gblaquiere-dev/eventsync**

*You can also build the image yourselves (the `Dockerfile` and the `cloudbuild.yaml` are here to help you)*

## Configuration

When you deploy a new Cloud Run service, you have to provide a configuration in the environment variable `CONFIG` (see
the deployment section for more details). The config must be in JSON format (see sample below)

### Configuration
```
{
  "serviceName": String
  "trigger": Trigger
  "endpoints":[Endpoint]
  "targetPubSub": TargetPubSub
}
```
Where
* `serviceName`: represent the name of the service. It will be also the name of the Firestore collection where are
  stored the events
* `trigger` is the definition of the `Trigger`
* `endpoints` is an array of Endpoint. The endpoint `eventKey` must be unique in the whole array
* `targetPubSub` is the PubSub target description, of type `TargetPubSub`

### Trigger
```
{
  "type": enum,
  "observationPeriod": int,
  "keepEventAfterTrigger": bool
}
```
Where
* `type` is the type of the trigger `none` or `window`
  * `none`: only API can trigger an event sync message, even if all the endpoints conditions are not yet met
  * `window`: even sync message generation is automatic when all the endpoints conditions are met. The condition's check
    is performed after each event reception.
* `observationPeriod` is the number of seconds in the past, from now, to retrieve the events when the endpoints 
 conditions are checked. The value is in seconds and must be > 0
* `KeepEventAfterTrigger` is a flag that indicates if the events must be flagged as exported or not after an event sync 
 message generation. This parameter is set to `false` by default. _See advanced feature for more details_

### Endpoints
```
{
  "eventKey": string
  "AcceptedHttpMethods": [string]
  "eventToSend": string
  "minNbOfOccurrence": int
}
```
Where
* `eventKey`: the name of the endpoint path to invoke from event source. The full path will be `/event/<eventKey>`.
The `eventKey` value must be unique in the whole list of configuration's `Endpoint`
* `AcceptedHttpMethods`: the list of accepted HTTP method for that `Endpoint`. The values can be GET, POST, OPTIONS, 
HEAD, PUT, DELETE, TRACE, CONNECT. If omitted, all the methods will be accepted
* `eventToSend`: define the event to add in the generated event sync message. Possible values are: ALL, FIRST, LAST,
BOUNDARIES. `ALL` is set by default (if missing).
  * `ALL`: all the event in the observation period are added to the event sync message
  * `FIRST`: only the first event in the observation period is added to the event sync message
  * `LAST`: only the latest event in the observation period is added to the event sync message
  * `BOUNDARIES`: only the first and the latest event in the observation period are added to the event sync message. If 
  there is only 1 event, it is not duplicated.
* `minNbOfOccurrence`: the minimal number of event to consider the endpoint as valid when a trigger check is performed.
The value must be > 0. If it is omitted or set to 0, it is set to 1 by default.

### TargetPubSub
```
{
  "topic": string
}
```
Where
* `topic`: PubSub topic to publish the message. The format must be the fully qualified topic name
  `projects/<ProjectID>/topics/<TopicName>`


### Sample and logs

At startup, the app summarizes the configuration in the logs. Check them out!

Here a sample configuration

```JSON
{
    "serviceName": "My First Event Sync",
    "trigger": {
        "type": "window",
        "observationPeriod": 3600,
        "keepEventAfterTrigger": false
    },
    "endpoints":[
        { 
          "eventKey": "entry1",
          "acceptedHttpMethods": ["POST"],
          "eventToSend": "ALL",
          "minNbOfOccurrence": 1
        },
        { "eventKey": "entry2"}
    ],
    "targetPubSub":{
        "topic": "projects/<ProjectID>/topics/<Topic>"
    }
}
```

## Deployment

Create the topic for the targetPubSub definition (if not yet exists)

```bash
gcloud pubsub topics create <TopicName>
```

Create a service account with the required permission:

```bash
# Create the service account
gcloud iam service-accounts create <ServiceAccount>

# The service account email will be <ServiceAccount>@<PROJECT_ID>.iam.gserviceaccount.com

# Grant the permissions
# Read/Write to Firestore
gcloud projects add-iam-policy-binding <PROJECT_ID> --member="serviceAccount:<ServiceAccountEmail>" --role="roles/datastore.user" --condition=None
# Create index in Firestore
gcloud projects add-iam-policy-binding <PROJECT_ID> --member="serviceAccount:<ServiceAccountEmail>" --role="roles/datastore.indexAdmin" --condition=None
# Publish message to PubSub topic (you can restrict this permission to the topic only if you want)
gcloud projects add-iam-policy-binding <PROJECT_ID> --member="serviceAccount:<ServiceAccountEmail>" --role="roles/pubsub.publisher" --condition=None
```

Set your configuration in a variable
```bash
export CONFIG='{
    "serviceName": "My First Event Sync",
    "trigger": {
        "type": "window",
        "observationPeriod": 3600,
        "keepEventAfterTrigger": false
    },
    "endpoints":[
        { 
          "eventKey": "entry1",
          "acceptedHttpMethods": ["POST"],
          "eventToSend": "ALL",
          "minNbOfOccurrence": 1
        },
        { "eventKey": "entry2"}
    ],
    "targetPubSub":{
        "topic": "projects/<ProjectID>/topics/<TopicName>"
    }
}'
```

Deploy to Cloud Run service (*here it is in `allow-unauthenticated` access for testing purposes. **Don't forget to secure the 
access in your real environments!***)
```bash
gcloud run deploy <CloudRunServiceName> \
  --image=gcr.io/gblaquiere-dev/eventsync \
  --allow-unauthenticated \
  --region=us-central1 \
  --platform=managed \
  --service-account=<ServiceAccountEmail> \
  --set-env-vars="^##^CONFIG=$CONFIG" 
```
*You can note here the special `^##^` to indicate gcloud CLI that the env var separator is no longer the comma `,` but the
`##` now. It prevents issues with JSON where comma is the standard attribute separator.*

The deployment display the `<CloudRunServiceUrl>`

## Test the service

Get the configuration

```bash
curl <CloudRunServiceUrl>/config
```

Add en event

```bash
curl -X POST -d "New test1" <CloudRunServiceUrl>/event/entry1
curl -X POST -d "New test2" <CloudRunServiceUrl>/event/entry2
```
*For an automatic trigger, with the current configuration and a trigger type set to `window`, you have to have at least
one event per endpoint. Check your PubSub!*

Manual trigger

```bash
curl <CloudRunServiceUrl>/event/trigger
```

You can also use the [Demo](https://github.com/guillaumeblaquiere/eventsync/tree/main/demo) section to test and
experiment with the service and the different configuration options.


## The output event sync message format

The output event sync JSON message as that format

### EventSync
```
{
  "eventID": string
  "date": date,
  "serviceName": string,
  "triggerType": enum,
  "events": map[string]EventList
}
```
Where
* `eventID` is the unique identifier of the ID based on a MD5 hash of all the messages in `events`. If 2 event sync are
generated with the same message, the ID will be the same and can help in subsequent deduplication
* `date` is the date of the generation of the event sync message
* `serviceName` is the name of the service provided in the configuration
* `triggerType` is an enum of the trigger type in the configuration: `none` or `windows`
* `events` is a map with, as key, the endpoints `eventKey` value, and an array of `EventList` as value

### EventList
```
{
  "firstEventDate": date,
  "lastEventDate": date,
  "numberOfEvents": int,
  "minNbOfOccurrence": int,
  "eventToSend": string
  "events": [Event]
}
```
Where
* `firstEventDate` is the date of the least recent event in the trigger's observation period. Not provided is no event 
are in  the `events` list
* `lastEventDate` is the date of the most recent event in the trigger's observation period. Not provided is no event
are in  the `events` list
* `numberOfEvents` is the number of events in the trigger's observation period. Can be different of the number of events
in the `events` list
* `minNbOfOccurrence` is the value set in the configuration to consider the endpoint valid
* `eventToSend` is the value set in the configuration to define the event to add in the generated event sync message
* `events` is the array of `Event` according to the configuration

### Event
```
{
  "datetime": date,
  "eventKey": string,
  "headers": map[string][string],
  "queryParams": map[string][string],
  "content": string
  "method": string
}
```
Where
* `datetime` is the date of the event reception by the application
* `eventKey` is the endpoint on which the event has been sent, represented by the eventKey
* `headers` represent the headers of the event HTTP request. It is a map with, as key, the entry, and as value an 
array of strings.
* `queryParams` represent the query parameters of the event HTTP request. It is a map with, as key, the entry, and as value an
  array of strings.
* `content` is the body content of the event HTTP request
* `method` is the HTTP method of the event HTTP request


### Sample 

Here a JSON sample
```JSON
{
  "date": "2023-01-12T11:06:20.908047236Z",
  "serviceName": "My First Event Sync",
  "triggerType": "window",
  "events": {
    "entry1": {
      "FirstEventDate": "2023-01-12T10:58:01.618171Z",
      "lastEventDate": "2023-01-12T10:58:01.618171Z",
      "numberOfEvents": 1,
      "minNbOfOccurrence": 1,
      "eventToSend": "ALL",
      "events": [
        {
          "datetime": "2023-01-12T10:58:01.618171Z",
          "eventKey": "entry1",
          "headers": {
            "Content-Type": [
              "application/x-www-form-urlencoded"
            ]
          },
          "queryParams": {
            "queryP": [
              "myQueryP"
            ]
          },
          "content": "New test1",
          "method": "POST"
        }
      ]
    },
    "entry2": {
      "FirstEventDate": "2023-01-12T11:06:19.432117Z",
      "lastEventDate": "2023-01-12T11:06:19.432117Z",
      "numberOfEvents": 1,
      "minNbOfOccurrence": 1,
      "eventToSend": "ALL",
      "events": [
        {
          "datetime": "2023-01-12T11:06:19.432117Z",
          "eventKey": "entry2",
          "headers": {
            "Content-Type": [
              "application/x-www-form-urlencoded"
            ]
          },
          "queryParams": {
            "queryP": [
              "myQueryP"
            ]
          },
          "content": "New test2",
          "method": "POST"
        }
      ]
    }
  }
}
```


# Advanced features

There are some advanced features to fine tune the behavior of the service

## KeepEventAfterTrigger

By default, when an event sync is triggered, all the messages are flagged as "already exported" and won't take into 
account in the subsequent operations:
* Check if all the endpoints conditions are met to generate a new event sync message. The conditions check take all the 
"valid" messages, i.e. all the messages in the observation period and **not** already exported
* Generate and event sync message with only the message used to check the endpoints conditions, i.e. all the messages 
in the observation period and **not** already exported

You can explicitly indicate to the service not to flag the messages to "already exported" to comply with your use case.

## Reset

If you want to clean the context, you can explicitly ask the application to flag all the events in the trigger's 
observation period. 

Like that, those messages won't be used to evaluate if the endpoints conditions are met to send a new event sync
message.

It's interesting in case of config change, or when a bug is fixed in the event sources to resume from a clean state and
context.

## Asynchronous post event processing

You can wish to keep a low latency in the event ingestion and to provide response ASAP to the event source. You have
the capacity to perform asynchronously the post event storage, _i.e. the trigger evaluation and the event sync message 
generation and sending (if applicable)_

That means only the event check and storage is performed synchronously and the request answer is sent. The rest of the
processing is performed in background.

Because you will need compute power OUTSIDE request handling, you have to use the 
[CPU Always ON](https://cloud.google.com/run/docs/configuring/cpu-allocation) on Cloud Run, *i.e. to
deactivate the CPU Throttling*

Natively, the app checks the current Cloud Run configuration. If the CPU Always On feature is activated, the post
process will be performed asynchronously.

However, for testing purpose, or to override that default behavior, you can force the behavior by setting the 
environment variable `ASYNC_EVENT_TRIGGER` to 
* `True` to force the asynchronous mode
* `False` to force the synchronous mode

*The case of the values does not matter*

Example of Cloud Run deployment with automatic configuration detection
```bash
gcloud run deploy <CloudRunServiceName> \
  --image=gcr.io/gblaquiere-dev/eventsync \
  --allow-unauthenticated \
  --region=us-central1 \
  --no-cpu-throttling \
  --platform=managed \
  --service-account=<ServiceAccountEmail> \
  --set-env-vars="^##^CONFIG=$CONFIG"
  
#Use --cpu-throttling to force the default Cloud Run behavior 
```

Example of Cloud Run deployment with force behavior
```bash
gcloud run deploy <CloudRunServiceName> \
  --image=gcr.io/gblaquiere-dev/eventsync \
  --allow-unauthenticated \
  --region=us-central1 \
  --cpu-throttling \
  --platform=managed \
  --service-account=<ServiceAccountEmail> \
  --set-env-vars="^##^CONFIG=$CONFIG##ASYNC_EVENT_TRIGGER=True"
```

## CORS deactivation

For demo purpose, it's required to deactivate the CORS. For that, add the env var `DISABLE_CORS` to anything. Example
```bash
gcloud run deploy <CloudRunServiceName> \
  --image=gcr.io/gblaquiere-dev/eventsync \
  --allow-unauthenticated \
  --region=us-central1 \
  --platform=managed \
  --service-account=<ServiceAccountEmail> \
  --set-env-vars="^##^CONFIG=$CONFIG##DISABLE_CORS=true"
```

# Contribution and local use

You can run locally the code with Golang installed on your environment and with the correct Google Cloud credential
loaded

For instance
```bash
# Use your own user credential
gcloud auth application-default login

# Use your own user credential to impersonate a service and use the service account permissions in the runtime
# Be sure your user account have the 'Service Account Token Creator' role granted on the project of the service account
# that you impersonate
gcloud auth application-default login --impersonate-service-account=<ServiceAccountEmail>
```

When you run the app locally, the ProjectID is not automatically detected from the runtime environment thanks to the
metadata servers. To solve that, you can set an environment variable `PROJECT_ID` with your project ID value.
You also must have your `CONFIG` as environment variable

```bash
# Put the config in the env vars
export CONFIG='{
    "serviceName": "My First Event Sync",
    "trigger": {
        "type": "window",
        "observationPeriod": 3600,
        "keepEventAfterTrigger": false
    },
    "endpoints":[
        { 
          "eventKey": "entry1",
          "acceptedHttpMethods": ["POST"],
          "eventToSend": "ALL",
          "minNbOfOccurrence": 1
        },
        { "eventKey": "entry2"}
    ],
    "targetPubSub":{
        "topic": "projects/<ProjectID>/topics/<TopicName>"
    }
}'

# Optionally set the asynchronous mode
export ASYNC_EVENT_TRIGGER="True"

PROJECT_ID=<YourProjectID> go run .
```

You can run the tests to ensure the non regression
```bash
go test ./...
```

## Build your own container

If you want to build your own container, you can use Cloud Build at the root directory of the project

```bash
gcloud builds submit --config=core/cloudbuild.yaml
```

And then deploy it, instead of the one provided.

Feel free to open issues for feature requests, code/documentation proposals and enhancement. ***Thanks!***

# Licence

This library is licensed under Apache 2.0. Full license text is available in
[LICENSE](https://github.com/guillaumeblaquiere/eventsync/tree/main/LICENSE).