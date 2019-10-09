---
title: operator-testing-tool
authors:
  - "@jmccormick2001"
  - "@joelanford"
reviewers:
  - TBD
  - "@joelanford"
approvers:
  - TBD
creation-date: 2019-09-13
last-updated: 2019-09-13
status: provisional|implementable|implemented|deferred|rejected|withdrawn|replaced
see-also:
  - "/enhancements/this-other-neat-thing.md"  
replaces:
  - "/enhancements/that-less-than-great-idea.md"
superseded-by:
  - "/enhancements/our-past-effort.md"
---

# operator-testing-tool


## Release Signoff Checklist

- \[ \] Enhancement is `implementable`
- \[ \] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created.

## Summary

This proposal is for a new Operator Testing Tool that validates Operator packaging as well as run simple functional test specific to the Operator’s APIs.

## Motivation

Operator Testing and Validation remains difficult today due to tooling sprawl, disparate definitions of validity, and the lack of means to provide custom tests per Operator in a way that is not tied to a particular CI pipeline.

Having high quality Operators is crucial to the success of the Operator Framework.

### Goals

 * Operator Developers can use a tool that reports on recommended, required and optional packaging requirements of a given bundle on disk
 * Operator Developers can use the same tool that reports on recommended, required and optional common black box tests of a deployed operator on cluster
 * Operator Developers can rely on a central source of truth of a valid Operator bundles shared among all Operator Framework components


### Non-Goals

not in scope: Operator Developers can use the same tool to run custom, functional tests of their Operator on cluster

## Proposal

### User Stories 

#### Story 1 - Show pass/fail in Scorecard Output
Today, the scorecard output shows a percentage of tests that were successful to the end user. This story is to change the scorecard output to show a pass or fail for each test that is run in the output instead of a success percentage. 

The exit code of scorecard would be 0 if all tests passed. The exit code would be non-zero if tests failed. With this change scores are essentially replaced with a list of pass/fail(s).

The scorecard would by default run all tests regardless if a single test fails.  Using a CLI flag such as below would cause the test execution to stop on a failure:
 * operator-sdk scorecard -l ‘testgroup=required’ --fail-fast

Tasks for this story include:
 * change scorecard by removing earnedPoints and maximumPoints from each test
 * change scorecard output by removing totalScorePercent, partialPass, earnedPoints, maximumPoints from scorecard output
 * change scorecard existing tests to remove partialPass, instead only returning a pass or fail
 * change scorecard CLI to return an exit code based on the pass/fail results from the set of executed tests 
 * (stretch goal) add support for --fast-fail CLI flag


Rough point estimate is 8 points.


#### Story 2 - Change Scorecard Test Selection
Today, the scorecard lets you select which tests are run by the plugins defined in the scorecard configuration (basic, olm). This story would change how users determine which tests are executed by scorecard.

This story would introduce labels for each test and then allow the scorecard CLI to filter which tests it runs based on label selectors.

Tests can fall into 3 groups: required, recommended, or optional. Tests can also be categorized as static or runtime. Labels for these groups and categories would allow a test to be more precisely specified by an end user.

Possible examples of specifying what tests to run are as follows: 
 * operator-sdk scorecard -l ‘testgroup in (required,recommended,optional)’ 
 * operator-sdk scorecard -l ‘testtype in (runtime,static)’
 * operator-sdk scorecard -l updatecrtest=true
 * operator-sdk scorecard -l someplugintest=true

Tasks for this story include:
 * Update the test interfaces to support arbitrary test-defined labels
 * Add the labels to the tests
 * Update the CLI and scorecard library to support label selectors to filter scorecard runs
 * A stretch goal would include the ability of the CLI to perform a dry-run of tests based on label selectors to show the user which tests would be executed for a given label selector value.

Rough point estimate for this story is 8 points.
#### Story 3 - Common Validation
The scorecard would use a common validation codebase to verify bundle contents. This story supports a single “source of truth” and avoids duplication or variance of validation logic. The scorecard user would be able to specify a bundle on the command line, that bundle would then be validated.  For example:
 * operator-sdk scorecard --bundle ./my-operator


Tasks for this story include:
 * determine the correct validation library to use
 * add support for a bundle directory flag to the CLI
 * change the scorecard code to use that validation library when validating a CR, CSV, or given a bundle directory
 * format the validation output to present to the CLI user in text or json

Rough point estimate in this story is 5 points.

#### Story 4 - Update scorecard Documentation
Tasks:
 * Document changes to CLI
 * Document new output formats
 * Document changes to configuration
 * Document breaking changes and removals



### Implementation Details/Notes/Constraints

A scheme for applying the labels to the tests would need to be developed as well as label names to categorize tests.

The “source of truth” for validation would need to be established such as which library the scorecard would use to perform CR/CSV/bundle validation.

If no runtime tests are specified, then the scorecard would only run the static tests and not depend on a running cluster.

### Risks and Mitigations

The scorecard would implement a version flag in the CLI to allow users to migrate from current functionality to the proposed functionality (e.g. v1alpha2):
 * operator-sdk scorecard --version v1alpha2


## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation and anything particularly
challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations).

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial proposal
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this
enhancement:
- Maturity levels - `Dev Preview`, `Tech Preview`, `GA`
- Deprecation

Clearly define what graduation means.

#### Examples

These are generalized examples to consider, in addition to the aforementioned
maturity levels.

##### Dev Preview -> Tech Preview

- Ability to utilize the enhancement end to end
- End user documentation, relative API stability
- Sufficient test coverage
- Gather feedback from users rather than just developers

##### Tech Preview -> GA 

- More testing (upgrade, downgrade, scale)
- Sufficient time for feedback
- Available by default

**For non-optional features moving to GA, the graduation criteria must include
end to end tests.**

##### Removing a deprecated feature

- Announce deprecation and support policy of the existing feature
- Deprecate the feature

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure this
is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

How will the component handle version skew with other components?
What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- During an upgrade, we will always have skew among components, how will this impact your work?
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI
  or CNI may require updating that component before the kubelet.

## Implementation History

Major milestones in the life cycle of a proposal should be tracked in `Implementation
History`.

## Drawbacks

The idea is to find the best form of an argument why this enhancement should _not_ be implemented.

## Alternatives

Similar to the `Drawbacks` section the `Alternatives` section is used to
highlight and record other possible approaches to delivering the value proposed
by an enhancement.

## Infrastructure Needed

Use this section if you need things from the project. Examples include a new
subproject, repos requested, github details, and/or testing infrastructure.

Listing these here allows the community to get the process for these resources
started right away.
