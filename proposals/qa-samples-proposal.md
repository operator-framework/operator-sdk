---
title: QA Samples Proposal
authors:
  - "@camilamacedo86"
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2019-11-22
last-updated: 2019-11-22
status: implementable
---

# QA Samples Proposal  

## Release Signoff Checklist

- \[x\] Enhancement is `implementable`
- \[x\] Design details are appropriately documented from clear requirements
- \[x\] Test plan is defined
- \[x\] Graduation criteria for dev preview, tech preview, GA
- \[x\] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]

## Summary

The Operator SDK has a repo with [sample projects](https://github.com/operator-framework/operator-sdk-samples
). This proposal describes quality improvements in for the sample projects.

## Motivation

- Address issues raised in the repository such as ; [Provide docker images for the samples](https://github.com/operator-framework/operator-sdk-samples/issues/88), [Add coveralls](https://github.com/operator-framework/operator-sdk-samples/issues/89), [Add unit test to cover the project](https://github.com/operator-framework/operator-sdk-samples/issues/87), [Add CI tests](https://github.com/operator-framework/operator-sdk-samples/issues/85).
- Make easier the process to review changes made for the projects
- Help users know how they can achieve common good practices

## Goals

- As a maintainer, I would like to ensure that samples projects continue to work after the changes performed in the PR are applied
- As an operator developer, I would like to know how to cover the projects with tests
- As an operator developer, I would like to know how to use the [test-framework][e2e-docs] to do unit and integration tests
- As an operator developer, I would like to know how to call the tests in the CI and integrate them with Travis
- As an operator developer, I would like to know good practices to ensure the quality of my operators projects 

**NOTE** The above goals are valid for the 3 types/projects Go, Ansible and Helm

### Non-Goals

- Change the code business logic implementation of the projects

## Proposal

- Cover the projects with unit and integration tests. 
- Integrate projects with Travis
- Integrate projects with Coveralls

### User Stories 

#### Unit tests for Go Memcached Sample

- I as a GO operator developer, I would like to know how to cover the projects with unit tests using the test-framework

**Acceptance Criteria** 
- The GO project should have a minimum of 70% of its implementation covered by unit tests. 
- The tests should all pass.
- The project should have a makefile target to call the tests
- A section with a short info over how to tests and the links for its documents in the README of the project
- The tests should be done using the default GO testing lib and the [test-framework][e2e-docs] provided by SDK
 
**NOTES** 
- See [here](https://github.com/dev4devs-com/postgresql-operator/blob/master/pkg/controller/database/controller_test.go) an example of how to cover the controller.
- See [here](https://github.com/dev4devs-com/postgresql-operator/blob/master/pkg/controller/database/fakeclient_test.go) an example of how to create the fake client. 

#### e2e tests for Go Memcached Sample

- I as a Go dev operator user, I'd like to know how to cover the projects with integration tests using the test-framework

**Acceptance Criteria** 
- The GO project should be covered with a few basic integration tests
- The tests should all pass.
- The project should have a makefile target to call the integration test
- A section with a short info over how to tests and the links for its documents in the README of the project
- The tests should be done using the default GO testing lib and the [test-framework][e2e-docs] provided by SDK

#### Unit tests for Ansible Memcached Sample

- As an Ansible operator developer, I would like to know how to cover the projects with unit tests using molecule

**Acceptance Criteria** 
- The Ansible project should be covered by tests using [molecule](https://github.com/operator-framework/operator-sdk-samples/tree/master/ansible/memcached-operator/molecule) which by default is scaffold
- The tests should all pass.
- A section with a short info over how to tests and the links for its documents in the README of the project
- The project should have a makefile target to call the tests

#### tests for Helm Memcached Sample

- As an Helm operator developer, I would like to how to cover the projects with tests

**Acceptance Criteria** 
- The Helm project should have test shell scripts as seen [here](https://github.com/operator-framework/operator-sdk/blob/master/hack/tests/e2e-helm.sh).- The tests should all pass.
- A section with a short info over how to tests and the links for its documents in the README of the project
- The project should have a makefile target to call the tests


#### e2e tests for Ansible Memcached Sample

- As an Ansible operator developer, I'd like to how to cover the projects with integration tests using molecule

**Acceptance Criteria** 
- The Ansible project should have a few integration tests using [molecule](https://github.com/operator-framework/operator-sdk-samples/tree/master/ansible/memcached-operator/molecule) which by default is scaffold
- The tests should all pass.
- A section with a short info over how to tests and the links for its documents in the README of the project
- The project should have a makefile target to call the tests

#### Travis Integration

- As an operator-sdk maintainer, I would like ensure that samples continue to work after the changes performed in the PR are applied.
- As an operator developer, I would like to know how to call the tests in the CI and integrate them with Travis

**Acceptance Criteria** 
- The Ansible, Helm and Go project should be integrated with Travis
- All PRs against the master branch should trigger the CI
- The unit and integration tests should be checked in the CI
- The CI should fail if one of the tests do not pass

#### Coveralls Integration

- As an operator developer, I'd like to know good practices to ensure the quality of my operators projects 

**Acceptance Criteria** 
- Ansible and GO Memcached samples should be integrated with [Coveralls](https://coveralls.io/)
- All above stories should be achieved successfully

### Risks and Mitigations

[Coveralls](https://coveralls.io/) may not work well with [molecule](https://github.com/operator-framework/operator-sdk-samples/tree/master/ansible/memcached-operator/molecule), if this is the case we can just not integrate with it or we can find a similar tool.

[operator-sdk-doc]:  https://sdk.operatorframework.io/
[e2e-docs]: https://sdk.operatorframework.io/docs/golang/legacy/e2e-tests/
