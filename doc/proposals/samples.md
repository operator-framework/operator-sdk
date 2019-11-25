---
title: Neat-proposals-Idea
authors:
  - "@camilamacedo86"
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2019-11-22
last-updated: 2019-11-22
status: implementable
see-also:
  - "/proposals/this-other-neat-thing.md"  
replaces:
  - "/proposals/that-less-than-great-idea.md"
superseded-by:
  - "/proposals/our-past-effort.md"
---

# Samples Quality Improvements 

## Release Signoff Checklist

- \[x\] Enhancement is `implementable`
- \[x\] Design details are appropriately documented from clear requirements
- \[x\] Test plan is defined
- \[x\] Graduation criteria for dev preview, tech preview, GA
- \[x\] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]

## Summary

This proposal is over some suggestions for we improve the quality of the Samples projects in https://github.com/operator-framework/operator-sdk-samples

## Motivation

- Address issues raised in the repository such as ; [Provide docker images for the samples](https://github.com/operator-framework/operator-sdk-samples/issues/88), [Add coveralls](https://github.com/operator-framework/operator-sdk-samples/issues/89), [Add unit test to cover the project](https://github.com/operator-framework/operator-sdk-samples/issues/87), [Add CI tests](https://github.com/operator-framework/operator-sdk-samples/issues/85).
- Make easier the process to review changes made for the projects
- Help users know how they can achieve common good practices

## Goals

- I as a maintainer, I'd like to ensure that the samples projects will still working after the changes performed in the PR easily
- I as a dev operator user, I'd like to know how to cover the projects with tests
- I as a dev operator user, I'd like to know how to use the test-framework to do unit and integration tests
- I as a dev operator user, I'd like to know how to how to call the tests in the CI and integrate them with Travis
- I as a dev operator user, I'd like to know good practices to ensure the quality of my operators projects 

**NOTE** The above goals are valid for the 3 types/projects Go, Ansible and Helm

### Non-Goals

- Change the code business logic implementation of the projects

## Proposal

- Cover the projects with unit and integration tests. 
- Integrate projects with Travis
- Integrate projects with Coveralls

### User Stories 

#### Unit tests for Go Memcached Sample

- I as a Go dev operator user, I'd like to know how to cover the projects with unit tests using the test-framework

**Acceptance Criteria** 
- The GO project should have at minimal 70% of its implementation covered by unit tests. 
- The tests should be working successfully
- The project should have a makefile target to call the tests
- The project should have an small section over how to test and with links for reference
- The tests should be done using the default go testing lib and the test-framework provided by SDK
 
**NOTES** 
- See [here](https://github.com/dev4devs-com/postgresql-operator/blob/master/pkg/controller/database/controller_test.go) an example over how to cover the controller.
- See [here](https://github.com/dev4devs-com/postgresql-operator/blob/master/pkg/controller/database/fakeclient_test.go) an example over how to create the fake client. 

#### e2e tests for Go Memcached Sample

- I as a Go dev operator user, I'd like to know how to cover the projects with integration tests using the test-framework

**Acceptance Criteria** 
- The GO project should be covered with a few basic integration tests
- The tests should be working successfully
- The project should have a makefile target to call the integration test
- The project should have an small section over how to test and with links for reference
- The tests should be done using the default go testing lib and the test-framework provided by SDK

#### Unit tests for Ansible Memcached Sample

- I as a Ansible dev operator user, I'd like to know how to cover the projects with unit tests using molecule

**Acceptance Criteria** 
- The Ansible project should be covered by tests using [molecule](https://github.com/operator-framework/operator-sdk-samples/tree/master/ansible/memcached-operator/molecule) which by default is scaffold
- The tests should be working successfully
- The project should have an small section over how to test and with links for reference
- The project should have a makefile target to call the tests

#### tests for Helm Memcached Sample

- I as a Helm dev operator user, I'd like to how to cover the projects with tests

**Acceptance Criteria** 
- The Helm project should have a few tests as using shell as it done [here](https://github.com/operator-framework/operator-sdk/blob/master/hack/tests/e2e-helm.sh).
- The tests should be working successfully
- The project should have an small section over how to test and with links for reference
- The project should have a makefile target to call the tests


#### e2e tests for Ansible Memcached Sample

- I as a Ansible dev operator user, I'd like to how to cover the projects with integration tests using molecule

**Acceptance Criteria** 
- The Ansible project should have a few integration tests using [molecule](https://github.com/operator-framework/operator-sdk-samples/tree/master/ansible/memcached-operator/molecule) which by default is scaffold
- The tests should be working successfully
- The project should have an small section over how to test and with links for reference
- The project should have a makefile target to call the tests

#### Travis Integration

- I as a maintainer, I'd like to ensure that the samples still working after the changes performed in the PR easily
- I as a dev operator user, I'd like to know how to call the tests in the CI and integrate them with Travis

**Acceptance Criteria** 
- The Ansible, Helm and Go project should be integrated with Travis
- All PR's made against the master should trigger th CI 
- The unit and integration tests should be checked in the CI
- The CI should fail if one of the tests do not pass

#### Coveralls Integration

- I as a dev operator user, I'd like to know good practices to ensure the quality of my operators projects 

**Acceptance Criteria** 
- Ansible and Go Memcached samples should be integrated with [Coveralls](https://coveralls.io/)
- All above stories should be achieved successfully

### Risks and Mitigations

May [Coveralls](https://coveralls.io/) do not work well with [molecule](https://github.com/operator-framework/operator-sdk-samples/tree/master/ansible/memcached-operator/molecule) and then, we can just not integrate it with or find another similar tool.

[operator-sdk-doc]:  ../../doc
