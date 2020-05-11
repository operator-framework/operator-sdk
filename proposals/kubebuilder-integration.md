---
title: Operator-SDK integrates Kubebuilder for Golang Operators
authors:
  - "@hasbro17"
  - "@estroz"
reviewers:
  - "@joelanford"
  - "@jmrodri"
  - "@dmesser"
approvers:
  - "@joelanford"
  - "@jmrodri"
  - "@dmesser"
creation-date: 2019-12-19
last-updated: 2020-03-05
status: implementable
see-also:
- https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/integrating-kubebuilder-and-osdk.md
- https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1.md
replaces: []
superseded-by: []
---

# Operator-SDK integrates Kubebuilder for Golang Operators

## Release Signoff Checklist

- \[x\] Enhancement is `implementable`
- \[x\] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]

## Summary

The Operator SDK should become compatible with Kubebuilder so that it has the
same project layout and CLI workflow for developing Go operators.
To achieve this goal without duplicating upstream code, the SDK should integrate
Kubebuilder’s CLI to reuse the scaffolding for project creation and lifecycle management.
At the same time the SDK should still be able to provide SDK specific features that work
with the new project layout.

In order for the SDK to wrap and reuse Kubebuilder while still maintaining the
SDK binary as the entrypoint, Kubebuilder should be extended with a CLI interface
that supports registering plugins. This lets the SDK reuse Kubebuilder’s existing
scaffolding code for Go operators, while still being able to extend the CLI for SDK
specific features/sub-commands and provide custom plugins that will allow the SDK to
scaffold different project types like Helm/Ansible in the future.


## Motivation

The Operator SDK and Kubebuilder largely provide the same experience for developing
Go operators since both projects defer to APIs provided by the same upstream projects.
The two projects differentiate in their CLI and UX of project setup, layout, and
lifecycle management. These differences are largely incidental of how the two
projects developed over time and not due to any use cases that are only addressed by either UX.

By adopting the same workflow and layout as Kubebuilder, the SDK gains better
alignment with the upstream community for writing Go operators. This allows the
SDK to be used interchangeably with Kubebuilder projects, which unifies the
experience of writing Go operators and resolves confusion on why the Operator SDK
and Kubebuilder differ in their UX to achieve the same goals. This lets the SDK
focus more on features that are outside the scope of Kubebuilder, such as
Operator Lifecycle Manager (OLM) integration and Scorecard.

Reusing the project scaffolding from Kubebuilder also reduces duplication in
the SDK. This frees up Operator SDK maintainers to focus more on collaborating
with the Kubebuilder maintainers on the upstream controller-runtime and controller-tools projects


## Goals

- Operator developers get the project layout and scaffolding for Golang Operators from Kubebuilder
- Operator developers can use the same command line switches as Kubebuilder to create and extend the project
  - This ensures Operator SDK binary compatiblity with existing Kubebuilder projects.
- An Operator Developer is notified about deprecation of `new --type=go` once `init` subcommand is in place.
- An Operator Developer should see the current UX maintained for these features: CSV generation, scorecard, and test-framework
- Operator developers can re-use an existing Kubebuilder project for Go-based Operators
- Operator developers uses the same Dockerfile and base image as Kubebuilder when using the upstream operator-sdk
- An Operator Developer has documentation that tells them how to use a different base image.
- An Operator Developer can leverage the Makefile from kubebuilder to instantiate the Operator locally
- Operator Developers can use cert-manager to generate TLS certificates for their webhook servers

### Non-Goals

- An Operator Developer can use the new plugin interface to scaffold and manage helm and ansible projects
  - While Kubebuilder’s new plugin based CLI architecture should allow the SDK to extend it for scaffolding
  Helm/Ansible projects, the implementation of those plugins is not currently in scope for this proposal.
- An Operator Developer can use the Makefile to run an Operator and is notified about the deprecation of `up local`
- Webhook support for Ansible/Helm based operators

## Proposal

### User Stories

#### Story 1 - Design a Kubebuilder plugin interface that lets the SDK reuse and extend the CLI

Status: **Done**

The complete proposal can be viewed [here][plugin-proposal].

After upstream discussions with the Kubebuilder maintainers there is consensus
that Kubebuilder should support an interface to register plugins that would be
used to provide the implementation of CLI commands. This way Kubebuilder would
not have to directly expose its CLI for the SDK’s consumption: the SDK would
reuse Kubebuilder’s plugins to scaffold new projects (`init`) and modify existing
ones (`create`) while also being able to register its own plugins to extend the
CLI for SDK specific subcommands (ex. `generate csv`, `scorecard`).

This plugin interface should also allow the SDK to provide custom plugin
implementations for the `init` and `create` subcommands so that it can customize
the scaffolding of Helm and Ansible projects.

There is a [proposal PR][plugin-proposal-pr] on Kubebuilder for the design of
such a plugin interface. The aim of this story is to address all reviewer
comments to achieve consensus on the plugin architecture and merge the proposal.

[plugin-proposal]: https://github.com/kubernetes-sigs/kubebuilder/blob/bf3667c/designs/extensible-cli-and-scaffolding-plugins-phase-1.md
[plugin-proposal-pr]: https://github.com/kubernetes-sigs/kubebuilder/pull/1250

#### Story 2 - Implement the Kubebuilder plugin interface and CLI pkg

After the plugin interface proposal has been accepted, the implementations of
the plugin interface and the CLI pkg should be added to Kubebuilder.

This involves writing plugin implementations for the following scaffolding
subcommands in Kubebuilder so that the SDK can register them for reuse in its own CLI:

- `kubebuilder init`
- `kubebuilder create api`
- `kubebuilder create webhook`

The goal of this story is to ensure a release of Kubebuilder that supports the
new plugin interface that the SDK can integrate downstream into its own CLI.


#### Story 3 - Integrate the Kubebuilder CLI into the SDK CLI to achieve the same workflow and project layout for Go operators

Once Kubebuilder supports plugin registration, the SDK CLI should be modified
to reuse Kubebuilder’s CLI and plugins so that the SDK workflow for developing
Go operators is identical to Kubebuilder’s workflow.

Project initialization and api/webhook scaffolding would be provided by the
upstream Kubebuilder CLI and plugins:  

- `operator-sdk new` ==> `operator-sdk init`
- `operator-sdk add api/controller` ==> `operator-sdk create api`
- `operator-sdk create webhook`

Other SDK subcommands can be removed and replaced by their equivalent Makefile
targets in the Kubebuilder workflow after ensuring that they fulfill all the same use cases.

- `operator-sdk generate k8s` ==> `make generate`
- `operator-sdk generate crds` ==> `make manifests`
- `operator-sdk build` ==> `make manager`, `make docker-build`

If any of the old subcommands above are invoked for a Go operator project, the
output should be a deprecation error that highlights the equivalent replacement
for it in the new Kubebuilder workflow.

A note on `operator-sdk run --local`: `kubebuilder` has a similar command `make run`
to run an operator locally which `operator-sdk` should . However there is enough
complexity in converting all functionality from `run --local` to `make run` that
doing so is out of scope as of this update. To align with the goals of this proposal,
both `operator-sdk run --local` and `make run` will be supported.

#### Story 4 - Update the SDK specific features and subcommands to work with the new project layout

Since features like the scorecard, CSV generation and test-framework have no
equivalent in the Kubebuilder workflow, those subcommands would be unchanged on the CLI.

- `operator-sdk generate csv`
- `operator-sdk scorecard`
- `operator-sdk test`

Where necessary the above commands should be adjusted to reflect the changes to
their expected input and output manifest paths that is consistent with the new project layout.


#### Story 5 - Update the Operator SDK e2e tests to work with the new project layout

The existing e2e tests and CI scripts for testing Go operators would need to be
updated to use the new layout so that CI passes for the new CLI workflow in the master branch.


#### Story 6 - Update the Go operator related documentation per the Kubebuilder workflow and project layout

The user documentation for Go operators such as the user-guide, CLI reference,
project layout, etc will need to be updated according to the new CLI and layout.
These doc updates must include a detailed explanation of Dockerfile usage in
Kubebuilder projects and how users can modify their vanilla Dockerfile to use
a parent UBI image.


### Implementation Details/Notes/Constraints

The integration work for the Go Operator CLI workflow can be done in the master
branch to avoid issues with merge conflicts from rebasing a separate branch at a later time.
The new CLI can be worked on behind a hidden subcommand, `operator-sdk alpha`, until it is ready to
replace the existing workflow. This would help avoid exposing it too early while
still providing the ability to test it on the master branch.


### Risks and Mitigations

- This proposal involves major breaking changes in all aspects of the project
that will discourage users from making the upgrade to the release involving these changes.
  - Besides having enough documentation to guide users to make the switch over
  to the new release, there should be some migration tooling to assist users in
  switching over their existing projects. **TODO:** These efforts will be
  addressed in a separate enhancement proposal.
- Once the SDK switches to using Kubebuilder for Go operators, the CLI UX for
Go vs Ansible/Helm operators will be fragmented. While they don’t target the
same users it could be confusing to see the operator-sdk binary support
different commands for initializing different project types.
  - With the proposed plugin architecture it should be possible to update the
  workflow for Helm/Ansible operators to be the same as Kubebuilder. However
  that is not currently in scope for this proposal.


[operator-sdk-doc]:  https://sdk.operatorframework.io/
