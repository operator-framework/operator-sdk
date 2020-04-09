---
title: README describing new command workflows for olm enabled operators
authors:
  - "@varshaprasad96"
reviewers:
  - "@jlanford"
  - "@dmesser"
  - "@estroz"
approvers:
  - "@jlanford"
  - "@dmesser"
  - "@estroz"
creation-date: 2020-04-09
last-updated: 2020-04-09
status: provisional
---

# README describing new command workflows for olm enabled operators

## Release Signoff Checklist

- \[ \] Enhancement is `provisional`
- \[ \] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created.

## Summary

This proposal provides documentation describing the possible workflows of new commands for generating operator manifests with user inputs and publishing operator bundle image for OLM to deploy the operator.

We plan to improve the developer experience for operator authors using Operator SDK with OLM by:
1. Providing an interactive subcommand to get inputs for generating CSV.
2. Providing a single command for generating operator manifests and for creating operator bundle image.

## Motivation

Currently, the `operator-sdk generate csv` command is used for generating CSVs, which sacffolds the project and writes the csv manifest on disk. This process still requires the developers to manually edit the CSV when any of the required fields are missing. Also, with the depreciation in the use of package manifests for olm, generating CSVs and creating operator bundle image does not require two separate commands. Integrating both the functionalities and providing a single command with an interactive input for collecting the required UI metadata to generate CSV as well as for building an operator bundle image will result in a better developer experience.

## Goals

The proposal is aimed at expalining the resulting process of developing an olm enabled operator from scratch with the above mentioned improvements.

## Proposal

### README

The following document discusses workflows for creating a new operator and running it with OLM:

Operator creation workflow:

1. Create a new operator project using the SDK Command Line Interface(CLI)
2. Define new resource APIs by adding Custom Resource Definitions(CRD)
3. Define Controllers to watch and reconcile resources
4. Write the reconciling logic for your Controller using the SDK and controller-runtime APIs
5. Use the SDK CLI to build and generate the operator deployment manifests

OLM Workflow:

1. Use `operator-sdk generate bundle` to generate kustomize base templates with CSV metadata, build and push the operator bundle image to the specified registry.
  * If the kustomize template having the operator and UI metadata is present, it is utilized along with the existing CRDs to build the operator image bundle.
  * If the kustomize template is not present, provide inputs regarding the UI metadata to the interactive prompts appearing further.
2. Use `operator-sdk run --olm` to generate kustomize base template with operator metadata, build and run the operator.

### Create and deploy an app-operator

```sh
# Create an app-operator project that defines the App CR.
$ mkdir -p $HOME/projects/example-inc/
# Create a new app-operator project
$ cd $HOME/projects/example-inc/
$ operator-sdk new app-operator --repo github.com/example-inc/app-operator
$ cd app-operator

# Add a new API for the custom resource AppService
$ operator-sdk add api --api-version=app.example.com/v1alpha1 --kind=AppService

# Add a new controller that watches for AppService
$ operator-sdk add controller --api-version=app.example.com/v1alpha1 --kind=AppService

# Set the username variable
$ export USERNAME=<username>

# Build and push the app-operator image to a public registry such as quay.io
$ operator-sdk build quay.io/$USERNAME/app-operator

# Login to public registry such as quay.io
$ docker login quay.io

# Push image
$ docker push quay.io/$USERNAME/app-operator

# Generate kustomize base templates with UI metadata and publish operator in a bundle image
# If you would like to write the operator manifests on disk, specify the path using the flag "--output-dir" 
$ operator-sdk generate bundle <IMAGE_REGISTRY>/<TAG>
      --package <Name of the package that bundle image belongs to>
      --kustomize-dir <Path to the directory containing kustomize base templates [default:./config/]>
      --output-dir <Optional output directory if operator manifests are to be written on disk>

# If the kustomize templates do not contain UI metadata for populating CSV, provide inputs for the interactive prompts that would appear further		
$ Provide DisplayName for your operator:		
$ Provide version for your operator:		
$ Provide minKubeVersion for CSV:		
...

# Deploy and test the operator on the cluster using OLM
# If path to the kustomize base templates is not present, they are generated from scratch inside ./config folder
$ operator-sdk run --olm --kustomize-dir <Path to directory containing kustomize base templates>
                   --operator-version <Version of the operator>
INFO[0000] loading packages
INFO[0000] generating manifests
...
NAME                            NAMESPACE    KIND                        STATUS
appservice.app.example.com    default      CustomResourceDefinition    Installed
app-operator.<version>        default      ClusterServiceVersion       Installed
...

# Cleanup
$ operator-sdk cleanup --olm --operator-version <Version of the operator>
```