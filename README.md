# Overview

EventSync offers a synchronization between different event sources and, when all the conditions are met, a new event is
triggered. It has been designed and tested on Cloud Run. _But it can be hosted on other Google Cloud runtime environment 
with no or a few updates_

It's especially interesting to synchronize discontinuous event sources, from different projects or providers.

*This [article](https://medium.com/google-cloud/eventsync-the-event-driven-management-missing-piece-baeb4fcb9315) 
presents that product and the problems it tackles*

# Project component

In this repository, you can find the [core](https://github.com/guillaumeblaquiere/eventsync/tree/main/core) part which 
contains the Eventsync product code and [documentation](https://github.com/guillaumeblaquiere/eventsync/tree/main/core/core.md)

You can also find the [Demo](https://github.com/guillaumeblaquiere/eventsync/tree/main/demo) which allows you to try 
and test the product. A simple web page is available and a demo backend is required to forward the Pubsub messages to 
the web page. You can find more detail in the [documentation](https://github.com/guillaumeblaquiere/eventsync/tree/main/demo/demo.md)


## Quick start

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

# Licence

This library is licensed under Apache 2.0. Full license text is available in
[LICENSE](https://github.com/guillaumeblaquiere/eventsync/tree/main/LICENSE).