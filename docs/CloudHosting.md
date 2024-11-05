# Hosting on the Internet

During its initial development, the backend was hosted on an iMac in someone's basement. While it "worked", this is not a sustainable practice for a number of reasons (such as, say, the hosting provider graduating from high school), as such the backend is now hosted on Google Cloud.

## Overview

The backend is hosted as a Google Cloud Run service. New versions of the backend are deployed via Cloud Build triggers watching for events on the GitHub repo. In addition, the service makes use of two object buckets for storage of both configuration and runtime files.

## Setup

As a pre-requisite, developers should walk through [Setup.md](Setup.md). In particular, setting up the Google Sheets API access will already involve some initial onboarding to Google Cloud.

Setting up the cloud hosting largely follows these guides on [continuous deployment from Git using Cloud Build](https://cloud.google.com/run/docs/continuous-deployment-with-cloud-build) and [deploying a new service on Cloud Run](https://cloud.google.com/run/docs/deploying#service). For budgetary reasons, it is recommended to use `us-central1` as the region for the cloud resources. A sample of the Service YAML manifest that gets generated is provided [here](../deploy/cloud-run-service.yaml) for reference.

The buckets have to be created manually through [Google Cloud Storage](https://cloud.google.com/docs/storage) before the service is deployed the first time. The buckets should be provisioned using the "Standard" storage class (to save on costs) and all other configurations can use their defaults. In the sample Service they are called `greenscout-backend-run` and `greenscout-backend-conf`, but they can be named whatever you want as long as they are selectable as volume mounts when configuring the containers.

It is **strongly recommended** that a service account is created specifically for use in the continuous deployment of the backend. All told, the service account used for the continuous deployment requires the following roles:

* [Cloud Build Service Account](https://cloud.google.com/iam/docs/understanding-roles#cloudbuild.builds.builder) (roles/cloudbuild.builds.builder)
* [Cloud Run Admin](https://cloud.google.com/iam/docs/understanding-roles#run.admin) (roles/run.admin)
* [Service Account User](https://cloud.google.com/iam/docs/understanding-roles#iam.serviceAccountUser) (roles/iam.serviceAccountUser)
* [Storage Object User](https://cloud.google.com/iam/docs/understanding-roles#storage.objectUser) (roles/storage.objectUser)

## Operations

The backend service should generally remain offline between robotics events. This means making sure the service is live and communicating with the frontend (*at least* a day prior to the event) as well as shutting it down after the event completes.

### Deployment

TODO

### Monitoring

TODO

### Shutdown

TODO
