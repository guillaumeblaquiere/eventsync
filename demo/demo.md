# Overview

The demo section offers a solution to test and visualize (through a web page) the behavior of Eventsync. It is based 
on the Eventsync product and a dedicated backend which listen PubSub and forward the messages to the web page.

# Deployment

There are 3 main pieces to deploy:

* The eventsync server
* The eventsync demo backend
* The eventsync demo frontend

## The eventsync server

You have to deploy the eventsync container with this configuration. *The PubSub topic has been created as mentioned in
the deployment part of Eventsync*

For the demo, you must deactivate the CORS to be able to run the frontend from any origin.

```bash
export CONFIG='{
    "serviceName": "My First Event Sync",
    "trigger": {
        "type": "window",
        "observationPeriod": 3600,
        "keepEventAfterTrigger": false
    },
    "endpoints":[
        { "eventKey": "entryA"},
        { "eventKey": "entryB"},
        { "eventKey": "entryC"}
    ],
    "targetPubSub":{
        "topic": "projects/<ProjectID>/topics/<TopicName>"
    }
}'


gcloud run deploy <CloudRunServiceName> \
  --image=gcr.io/gblaquiere-dev/eventsync \
  --allow-unauthenticated \
  --region=us-central1 \
  --platform=managed \
  --service-account=<ServiceAccountEmail> \
  --set-env-vars="^##^CONFIG=$CONFIG##DISABLE_CORS=true"
```

## The Eventsync demo backend

You first need to create a pull subscription on the Eventsync 

```bash
gcloud pubsub subscriptions create eventsync-demo \
  --topic <TopicName>
```

Build the Eventsync backend container for the demo from the `demo/backend` directory

```bash
cd demo/backend
gcloud builds submit 
```

Create a service account with the required permission:

```bash
# Create the service account
gcloud iam service-accounts create <ServiceAccount>

# The service account email will be <ServiceAccount>@<PROJECT_ID>.iam.gserviceaccount.com

# Grant the permissions
# Subscribe only to the created subscription
gcloud pubsub subscriptions add-iam-policy-binding eventsync-demo \
  --member="serviceAccount:<ServiceAccountEmail>" --role="roles/pubsub.subscriber"
```

Deploy the backend container
```bash
gcloud run deploy <eventsyncDemo service name> \
  --image=gcr.io/gblaquiere-dev/eventsync-demo \
  --allow-unauthenticated \
  --region=us-central1 \
  --platform=managed \
  --service-account=<ServiceAccountEmail> \
  --timeout=10m \
  --set-env-vars="SUBSCRIPTION=eventsync-demo"  
```

## The Eventsync demo frontend

In the `index.html` file, at the top of the file, you can change the backend URL to point to the backend you deployed.

```
let demoBackendUurl = "<eventsyncDemo service URL>";
let eventsyncUrl = "<eventsync service URL>"
```

Open the `index.html` in your browser. You can also use an [online viewer](http://htmlpreview.github.io/?https://github.com/guillaumeblaquiere/eventsync/blob/main/demo/frontend/index.html)

*You can also update those values directly in the web page (in the bottom, and click `APPLY`)*

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
You also must have your PubSub pull `SUBSCRIPTION` name as environment variable

```bash
# Navigate to the backend directory
cd demo/backend

SUBSCRIPTION=<YourSubscription (eventsync-demo)> PROJECT_ID=<YourProjectID> go run .
```

## Build your own container

If you want to build your own container, you can use Cloud Build at the root directory of the project

```bash
gcloud builds submit --config=demo/backend/cloudbuild.yaml
```

And then deploy it, instead of the one provided.

# Licence

This library is licensed under Apache 2.0. Full license text is available in
[LICENSE](https://github.com/guillaumeblaquiere/eventsync/tree/main/LICENSE).