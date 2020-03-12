---
title: Ansible/Helm add api command proposal for Operator SDK
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
last-updated: 2020-03-12
status: implementable
---


# Ansible/Helm add api command proposal for Operator SDK


## Release Signoff Checklist

- \[x\] Enhancement is `implementable`
- \[ \] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Accceptance criteria
- \[ \] User-facing documentation is created



## Summary

The proposal is to enable Ansible/Helm operator users to create additional APIs, through SDK CLI.

## Motivation

Ansible/Helm operator users are not able to create additonal APIs via CLI, once the original project scaffolds. Today, users have to manually add necessary files to the project scaffold.

## Goals

* Ansible/Helm operator developer can use existing SDK CLI commands to create additonal APIs as needed.
* Ansible/Helm operator developer can find supported documentation for the same.

## Proposal

### User Stories 

#### Story 1 - Ansible operator additional API
As an  Ansible operator developer, I would like to scaffold additional api, once the original Ansible operator project has been created, using following command.
`operator-sdk add api --kind <kind> --api-version <group/version>`

##### Implementation Details/Notes/Constraints

* Analyze how existing Go operator is scaffolding additonal APIs.
* Manually add additonal API for an Ansible operator example as PoC.
* `operator-sdk new memcached-operator --api-version=cache.example.com/v1alpha1 --kind=Memcached --type=ansible` scaffolds new ansible based operator for the user with given API.
* Once scaffolded the goal is to use below command to create additonal API.
  `operator-sdk add api --api-version=cache.example.com/v1alpha1 --kind=Sample`
* Note that in proposed command, we are not using `--type=ansible` to denote the operator type. To achieve this, we can use already existing function,
  `func IsOperatorAnsible()  bool` from [here][isoperatoransible].
* The goal is to scaffold below files for each additional API resource:  
    * Update ansible role at `../deploy/role.yaml` to add new `--api-version` if provided. If not, add new `--kind=Sample` to existing `apiGroups`
    * Create directory for new `--kind` under `../roles/`
    * CRD and CR files are generated at `../deploy/crds/`                                   
    * watches.yaml at `../watches.yaml` gets updated with new API resource.
    * Update playbook with the generated resources (**TBD**)
    * Update Molecule tests under `../molecule/` (**TBD**)

##### Acceptance Criteria

* Documentation is updated with steps to add additonal APIs for Ansible based operator.
* Flags options available for `operator-sdk new`, should also be made available for `operator-sdk add api`
* Ansible operator developer should be able to scaffold all resources needed for the additional API with following command
  `operator-sdk add api --kind <kind> --api-version <group/version>` 


#### Story 2 - Helm operator additional API
As Helm operator developer, I would like to scaffold additional API, once the original Helm operator project has been created, using following command.
`operator-sdk add api --kind <kind> --api-version <group/version>` with optional `--helm-chart`flag.

##### Implementation Details/Notes/Constraints

* Analyze how existing Go operator is scaffolding additonal APIs.
* Manually add additonal api for Helm operator example as PoC.
* `operator-sdk new memcached-operator --api-version=cache.example.com/v1alpha1 --kind=Memcached --type=helm` scaffolds new helm based operator for the user with given API.
* Once scaffolded the goal is to use below command to create additonal API.
  `operator-sdk add api --api-version=cache.example.com/v1alpha1 --kind=Nginx`
* Note that in proposed command, we are not using `--type=helm` to denote the operator type. To achieve this, we can use already existing function,
  `func IsOperatorHelm()  bool` from [here][isoperatorhelm].
* The goal is to scaffold below files for each additional API resource:  
    * Update helm role at `../deploy/role.yaml` to add new `--api-version` if provided. If not, add new `--kind=Nginx` to existing `apiGroups`
    * CRD and CR files are generated at `../deploy/crds/`
    * Charts directory contanins new folder for new chart provided using `--helm-chart` flag                              
    * watches.yaml at `../watches.yaml` gets updated with new API resource.

##### Acceptance Criteria

* Documentation is updated with steps to add additonal APIs for Helm based operator.
* Flags options available for `operator-sdk new`, should also be made available for `operator-sdk add api`
* Helm operator developer should be able to scaffold all resources needed for the additional API with below command, `--helm-chart` being optionl
`operator-sdk add api --kind <kind> --api-version <group/version> --helm-chart=stable/nginx`


[isoperatoransible]:https://github.com/operator-framework/operator-sdk/blob/master/internal/util/projutil/project_util.go#L188
[isoperatorhelm]:https://github.com/operator-framework/operator-sdk/blob/master/internal/util/projutil/project_util.go#L193
