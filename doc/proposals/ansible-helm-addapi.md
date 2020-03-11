---
title: Ansible/Helm add api command  Proposal for Operator SDK
authors:
  - "@bharathi-tenneti"
reviewers:
  - "@fabianvf"
  - "@cmacedo"
approvers:
  - "@fabianvf"
  - "@jlanford"
  - "@dmesser"
creation-date: 2020-03-10
last-updated: 2020-03-11
status: implementable
---


# Ansible/Helm add api command  Proposal for Operator SDK


## Release Signoff Checklist

- \[x\] Enhancement is `implementable`
- \[x\] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Accceptance criteria
- \[ \] User-facing documentation is created



## Summary

In current scenario, users are not able to add additional APIs for Ansible/Helm based opertators using SDK CLI. This change will make it easier to add additional apis using CLI for Ansible and Helm based operators.

## Motivation

For Ansible/Helm operator SDK, user is not able to add additional apis for already scaffolded operators.

## Goals

The goal of the proposal if implemented, is to enable users add additional APIs once the original Ansible/Helm operator is scaffolded.

## Non-Goals
NA

## Proposal

### User Stories 

#### Story 1 - Ansible Operator-sdk
Enable Ansible operator-sdk to scaffold additional api, once the oroginal Ansible operator project has been created, using bwloq command.
`operator-sdk add api --kind <kind> --api-version <group/version>`  

##### Tasks included:

* Update ansible role at ../deplo/role.yaml
* Create directpry for new kind under ../roles/
* CRD and CR files are generated at ../deploy/crds/                                   
* Update watches file at ../watches.yaml
* Update playbook with the generated resources (**TBD**)
* Update Molecule tests under ../molecule/ (**TBD**)

#### Story 2 - Helm Operator-sdk
Enable Helm operator-sdk to scaffold additional api, once the original Helm operator project has been created, using below command, --helm-chart being optional.
`operator-sdk add api --kind <kind> --api-version <group/version> --helm-chart=stable/nginx` 

##### Tasks included:

* Update helm role at ../deplo/role.yaml
* CRD and CR files are generated at ../deploy/crds/ 
* Charts directory is generated under ../helm-charts
* Update watches file at ../watches.yaml

### Implementation Details/Notes/Constraints
* Analyze how existing Go operator is scaffolding additonal apis.
* Manually add additonal api for an Ansible operator example as PoC.
* Modify existin "add api" functionality under /cmd/add/api.go, to include ansible workflow.
* Discuss on whether to generate playbook for ansible in case of additional api.
* Discuss on whether to generate molecule tests as well for additional apis.


### Risks and Mitigations
NA

### Acceptance Criteria

* Documentation is updated with steps to add additonal apis for both Ansible and Helm.
* Flags options which are related to the API scaffold and available for operator-sdk new project --type   also should be provided in the operator-sdk new api for the types
* Ansible User should be able to scadffold all resources needed for the additional api with below command
`operator-sdk add api --kind <kind> --api-version <group/version>` 
* Helm User should be able to scadffold all resources needed for the additional api with below command,   --helm-chart being optionl
`operator-sdk add api --kind <kind> --api-version <group/version> --helm-chart=stable/nginx` 









