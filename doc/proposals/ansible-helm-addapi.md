# Ansible/Helm add api command  Proposal for Operator SDK

> Status: implementable
> 

- [Background](#background)
- [Goals](#goals)
- [Commands](#commands)
- [Ansible_Changes] (#ansible changes)
- [Helm_Changes] (#helm changes)

## Background

In current scenario, users are not able to add additional APIs for Ansible/Helm based opertators using SDK CLI.

## Goals

The goal of the proposal if implemented, is to enable users add additional APIs once the original Ansible/Helm operator is scaffolded.

## Commands
We are updating existing commands to accommodate the ansible/helm operator.  Changes to the `cmd\add\api.go` is needed.

`operator-sdk add api --kind <kind> --api-version <group/version>`     

## Ansible Changes

* Udate ansible role at ../deplo/role.yaml
* CRD and CR files are generated at ../deploy/crds/                                   
* Update watches file at ../watches.yaml
* Update playbook with the generated resources(TBD)

## Helm Changes

* Update helm role at ../deploy/role.yaml
* CRD and CR files are generated at ../deploy/crds/
* Charts directory is generated under ../helm-charts
* Update watches file at ../watches.yaml



