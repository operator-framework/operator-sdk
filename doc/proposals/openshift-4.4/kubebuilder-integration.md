---
title: OpenShift 4.4: Operator-SDK integrates Kubebuilder for Golang Operators
authors:
  - "@hasbro17"
reviewers:
  - "@estroz"
  - "@joelanford"
approvers:
  - "@estroz"
  - "@joelanford"
creation-date: 2019-12-19
last-updated: 2019-12-19
status: implementable
see-also:
  - https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/integrating-kubebuilder-and-osdk.md
replaces: []
superseded-by: []
---

# OpenShift 4.4: Operator-SDK integrates Kubebuilder for Golang Operators

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

- Operator SDK projects should have the same project layout as Kubebuilder projects for Go operators
- The Operator SDK CLI and workflow for extending (adding APIs and webhooks) and managing the lifecycle (generate manifests, build and run) of a project should be the same as Kubebuilder
  - This ensures Operator SDK binary compatiblity with existing Kubebuilder projects.
- Operator SDK specific features such as CSV generation, scorecard and test-framework should work the same in the new project layout.
- Operator SDK developers can use cert-manager to generate TLS certificates for their webhook servers


### Non-Goals

- Changing the SDK scaffolding and workflow for Helm and Ansible operators to align with Kubebuilder’s CLI
  - While Kubebuilder’s new plugin based CLI architecture should allow the SDK to extend it for scaffolding Helm/Ansible projects, the implementation of those plugins is not currently in scope for this proposal.
- Webhook support for Ansible/Helm based operators


## Proposal

### User Stories

#### Story 1 - Design a Kubebuilder plugin interface that lets the SDK reuse and extend the CLI

After upstream discussions with the Kubebuilder maintainers there is consensus 
that Kubebuilder should support an interface to register plugins that would be 
used to provide the implementation of CLI commands. This way Kubebuilder would 
not have to directly expose its [cobra][cobra] CLI for the SDK’s consumption, and the SDK 
could still reuse Kubebuilder’s plugins to scaffold new projects (init and create) 
while also being able to register its own plugins to extend the CLI for SDK specific 
subcommands (csv-gen, scorecard).

This plugin interface should also allow the SDK to provide custom plugin 
implementations for the init and create subcommands so that it can customize the 
scaffolding of Helm and Ansible projects in the future.

There is a [proposal][plugin-proposal] on Kubebuilder for the design of such a plugin interface. 
The aim of this story is to address all reviewer comments to achieve consensus 
on the plugin architecture and merge the proposal.

[plugin-proposal]: https://github.com/kubernetes-sigs/kubebuilder/pull/1250
[cobra]: https://github.com/spf13/cobra


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

Other SDK subcommands can be removed and replaced by their equivalent Makefile targets in the Kubebuilder workflow after ensuring that they fulfill all the same use cases.

- `operator-sdk generate k8s` ==> `make generate`
- `operator-sdk generate crd` ==> `make manifests`
- `operator-sdk build` ==> `make manager`, `make docker-build`
- `operator-sdk up local` ==> `make run`

If any of the old subcommands above are invoked for a Go operator project, the 
output should be a deprecation error that highlights the equivalent replacement 
for it in the new Kubebuilder workflow.


#### Story 4 - Update the SDK specific features and subcommands to work with the new project layout

Since features like the scorecard, CSV generation and test-framework have no 
equivalent in the Kubebuilder workflow, those subcommands would be unchanged on the CLI.

- `operator-sdk gen-csv`
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


### Implementation Details/Notes/Constraints
The integration work for the Go Operator CLI workflow can be done in the master 
branch to avoid issues with merge conflicts from rebasing a separate branch at a later time.
The new CLI can be worked on behind a hidden subcommand, `operator-sdk alpha`, until it is ready to 
replace the existing workflow. This would help avoid exposing it too early while 
still providing the ability to test it on the master branch. 


### Risks and Mitigations

- This proposal involves major breaking changes in all aspects of the project that will discourage users from making the upgrade to the release involving these changes.
  - Besides having enough documentation to guide users to make the switch over to the new release, there should be some migration tooling to assist users in switching over their existing projects. **TODO:** These efforts will be addressed in a separate enhancement proposal.
- Once the SDK switches to using Kubebuilder for Go operators, the CLI UX for Go vs Ansible/Helm operators will be fragmented. While they don’t target the same users it could be confusing to see the operator-sdk binary support different commands for initializing different project types.
  - With the proposed plugin architecture it should be possible to update the workflow for Helm/Ansible operators to be the same as Kubebuilder. However that is not currently in scope for this proposal.


[operator-sdk-doc]: ../../../doc