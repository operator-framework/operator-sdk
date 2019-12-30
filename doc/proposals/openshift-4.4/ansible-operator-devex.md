---
title: Ansible Operator Developer Experience Improvements
authors:
  - "@fabianvf"
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2019-12-18
last-updated: 2019-12-18
status: implementable
see-also:
  - https://github.com/operator-framework/operator-sdk/pull/2048
replaces: []
superseded-by: []
---

# OpenShift 4.4: Ansible Operator Developer Experience Improvements

## Release Signoff Checklist

- \[x\] Enhancement is `implementable`
- \[x\] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]

## Open Questions (optional)

This is where to call out areas of the design that require closure before deciding
to implement the design.  For instance, 
 > 1. This requires exposing previously private resources which contain sensitive
  information.  Can we do this? 

## Summary

During the OpenShift 4.4 development timeframe, we plan to improve the developer
experience, focusing on a few key areas:

1. Ease development/debugging friction by making it easier to debug and test
    locally.
2. Allow users to customize the behavior of the proxy, to allow them to work around
    APIs that do not follow conventions rather than requiring additional logic in
    the Golang controller.
3. Update scaffolding and examples to take new tooling and best practices into
    consideration.

## Motivation

The motivation for these changes is to ease development friction and give more
power to the users, shortening the development iteration loop and preventing them
from being blocked by the Golang portion of the Ansible Operator due to
Kubernetes/OpenShift API inconsistencies or operator-sdk bugs/behaviors.

## Goals

1. Allow users to whitelist/blacklist resources to be passed through the cache
2. Update Ansible scaffolding, test scenarios, and examples to make use of newer
    Ansible features and best practices, including support for collections and
    simpler and more flexible test scenarios.
3. Readable logs without additional dependencies, and ideally without a sidecar 
    container.

### Non-Goals

- Perfection of the logging. We are trying to get incremental improvement for the
    logs of the Ansible Operator, but there will likely be additional changes and
    improvements that we make before a 1.0 release.
- Significant changes to the tooling used for tests. We are aiming primarily to
    remove/refactor unnecessarily complicated aspects of the scaffolded tests,
    while making it easier for the user to get the behavior that they need without
    needing to delve into some of the nittier aspects of the testing logic.

## Proposal

This is where we get down to the nitty gritty of what the proposal actually is.

### User Stories

#### Story 1 - Use operator-sdk up local to run the Ansible-based Operator

As a user, I would like to be able to run `operator-sdk up local` to run my
Ansible-based Operator without requiring additional work or options. Although 
it is currently possible to use `up local`, there are a variety of limitations
that make the experience inferior to deploying the operator to a real cluster.

1. I would like to be able to run `up local`, and see both the operator logs as
    well as the logs from the Ansible stdout. I would like both logs to be useful
    and readable.
1. I would like to be able to run `up local` and not need to change my
    `watches.yaml` to reflect the different paths of my host vs the operator
    container.

#### Story 2 - Use the molecule scenarios to test in a variety of environments

As a user, I would like to be able to use the scaffolded molecule scenarios to 
run the same set of tests against Kubernetes clusters in the following scenarios:

1. An ephemeral cluster provisioned by molecule
1. An existing cluster local to my machine
1. An existing cluster in the cloud
1. A cluster with OLM installed
1. A cluster without OLM installed


#### Story 3 - Customize the cache to work around API issues

As a user, if there are issues in the way the proxy/cache handles a certain resource,
I should be able to work around those issues without needing to wait for bugfixes or
features in the operator-sdk, by preventing those resources from being passed through
the cache at all. For example, if I need to access an OpenShift Project resource (which
is not cacheable, because it is not watchable), I should be able to specify that the 
Project resource skips the cache in my watches.yaml.


#### Story 4 - Readable logs by default

As a user, I would like to be able to tail the log of the primary operator pod and be
able to see both what the Golang process and the child Ansible processes are doing,
in a human readable way.


### Risks and Mitigations

Because these are primarily UX improvements, there isn't too much risk of breaking
compatibility or hitting unforeseen issues. The primary issue I could see is that
if we begin parsing the event messages that Ansible gives us we become reliant on
their structure, which is not guaranteed to remain static, so it could lead to a
marginally higher maintenance burden when changing Ansible versions.

## Design Details

### Test Plan

1. We will add at least one test for each molecule scenario (when possible). 
1. We will add tests that make Ansible emit messages of each type that it supports,
    to help ensure that we can easily catch changes to the messag format.
1. We will add a test scenario that uses `up local`.

### Graduation Criteria

N/A

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Implementation History

Major milestones in the life cycle of a proposal should be tracked in `Implementation
History`.

## Drawbacks

None

## Alternatives

None

[operator-sdk-doc]:  ../../../doc
