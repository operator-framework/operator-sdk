---
title: OpenShift 4.4: Operator SDK Tech Debt
authors:
  - "@joelanford"
reviewers:
  - "@camilamacedo86"
  - "@estroz"
approvers:
  - TBD
creation-date: 2019-12-18
last-updated: 2019-12-18
status: implementable
see-also: []
replaces: []
superseded-by: []
---

# OpenShift 4.4: Operator SDK Tech Debt

## Release Signoff Checklist

- \[x\] Enhancement is `implementable`
- \[x\] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]

## Summary

During the OpenShift 4.4 development timeframe, there are two primary areas of
the Operator SDK project that we plan to improve.

1. Introduction of unobtrusive code health improvements (Go linting, test coverage)
2. UX and code improvements to the `generate` subcommand.

## Motivation

The motivation for these changes is to improve the user experience of the SDK's
command line interface and to improve code health to ensure a quality product,
increase developer productivity, and provide a more inviting project for external
contributors.

## Goals

1. Enable `golangci-lint` in the project Makefile and in CI, reduce the number of
   linter errors, and permenantly enable more linters to keep avoid introducing
   new lint errors.
2. Enable code coverage metrics for unit tests and push results to coveralls.io
   during CI to help contributors and maintainers keep code quality high and
   reduce the likelihood of regressions.
3. Refactor the `generate` subcommand to improve user experience, reduce confusion,
   and simplify the codebase.

### Non-Goals

There are numerous other areas of the codebase that are in need of refactoring
(e.g. scorecard and the test framework). Due to the size and scope of these
other refactorings, they need to be broken out and handled on their own, either
in the context of another epic or as a separate epic altogether. Therefore, this
epic is limited in the scope to the specific refactorings and improvements
discussed above.

It is also not in the scope of this epic to actually _improve_ test coverage. This
epic's goal is to simply make test coverage metrics available and visible.

## Proposal

This is where we get down to the nitty gritty of what the proposal actually is.

### User Stories

#### Story 1 - Integrate `golangci-lint` into the development and CI processes

Go linting will use golangci-lint. It be introduced as a `make` target and as a step
in the sanity tests that run as part of the SDK's continuous integration test suite.
The `make` target will run all of golangci-lint's linters, whereas the sanity test
will run a subset that we expect to pass. Over the course of the 4.4 development
cycle, work will continue to fix outstanding lint issues so that more linters can
be enabled in CI.

#### Story 2 - Instrument unit tests with code coverage

Test coverage will be tracked by running unit tests with code coverage enabled and 
submitting results to coveralls.io during CI. This integration will highlight code
coverage improvements or regressions in each PR, which will incentivize contributors
and reviewers to improve code coverage over time.

#### Story 3 - Refactor CRD generation

Improvements to the generate subcommand will include the deprecation and
removal of the `generate openapi` subcommand (to be replaced with the `generate
crds` subcommand) and the internal refactoring of the `generate crds` subcommand
to use a more flexible entrypoint into the underlying `controller-tools` package
that actually implements the CRD generation.

The `generate openapi` command currently generates both CRD yaml files and Go code
that defines the OpenAPI structs for the operator's Go API types. However, very few
use cases require the generated OpenAPI code. By removing the Go OpenAPI code
generation, we are left with just CRD generation, hence the need for an improved
name (`generate crds`).

We will add the new `generate crds` subcommand and deprecate `generate openapi`. A
deprecation message will be included that directs users who need to continue
generating Go OpenAPI code how to do it with the upstream `openapi-gen` tool
directly.

### Risks and Mitigations

Only risk is that deprecation of `generate openapi` will break a small subset of SDK
users. The mitigation is that the SDK will include instructions for running code 
generation tools directly, which will be a 1-to-1 replacement of the deprecated
functionality in Operator SDK.

## Design Details

### Test Plan

All unit tests will gain coverage metrics, which will increase visibility on the areas
of the codebase that do not have adequete testing. 

The `generate` subcommand changes will also result in refactored unit and e2e tests,
which will no longer need to check for the existence of generated Go OpenAPI code.

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

## Infrastructure Needed (optional)

None

[operator-sdk-doc]:  ../../../doc
