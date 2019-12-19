---
title: OpenShift 4.4: Operator SDK supports Kubernetes 1.17
authors:
  - "@joelanford"
reviewers:
  - TBD
approvers:
  - TBD
creation-date: 2019-12-19
last-updated: 2019-12-19
status: implementable
see-also:
  - https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.17.md#changes
replaces: []
superseded-by: []
---

# OpenShift 4.4: Operator SDK supports Kubernetes 1.17

This is the title of the enhancement. Keep it simple and descriptive. A good
title can help communicate what the enhancement is and should be considered as
part of any review.

To get started with this template:
1. **Pick a domain.** Find the appropriate domain to discuss your enhancement.
1. **Make a copy of this template.** Copy this template into the directory for
   the domain.
1. **Fill out the "overview" sections.** This includes the Summary and
   Motivation sections. These should be easy and explain why the community
   should desire this enhancement.
1. **Create a PR.** Assign it to folks with expertise in that domain to help
   sponsor the process.
1. **Merge at each milestone.** Merge when the design is able to transition to a
   new status (provisional, implementable, implemented, etc.). View anything
   marked as `provisional` as an idea worth exploring in the future, but not
   accepted as ready to execute. Anything marked as `implementable` should
   clearly communicate how an enhancement is coded up and delivered. If an
   enhancement describes a new deployment topology or platform, include a
   logical description for the deployment, and how it handles the unique aspects
   of the platform. Aim for single topic PRs to keep discussions focused. If you
   disagree with what is already in a document, open a new PR with suggested
   changes.

The `Metadata` section above is intended to support the creation of tooling
around the enhancement process.

## Release Signoff Checklist

- \[x\] Enhancement is `implementable`
- \[x\] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created in [operator-sdk/doc][operator-sdk-doc]

## Summary

The release of Kubernetes 1.17 brings new features, enhancements, and bug fixes
to the Kubernetes API and Kubernetes libraries that support operator development.
The focus of this enhancement is to bring Kubernetes 1.17 support to Operator SDK.

See the [Kubernetes 1.17.0 CHANGELOG][changelog] for details.

[changelog]: https://github.com/kubernetes/kubernetes/blob/master/CHANGELOG-1.17.md#changes

## Motivation

The motivation for this enhancement is to give operator developers access to the
latest Kubernetes features and so that existing operator projects have a
continued upgrade path to ensure their compatibility with the latest versions of
Kubernetes.

## Goals

The goal is to update the Kubernetes dependencies of the Operator SDK to use 1.17.

### Non-Goals

N/A

## Proposal

### User Stories

#### Story 1

As an operator developer, I want to take advantage of the features of Kubernetes
1.17 and ensure that my operator is compatible with any API changes and removals
in Kubernetes 1.17.

### Risks and Mitigations

The biggest risk with this proposal, as always, is that Operator SDK depends on
other projects that depend on Kubernetes, so it is typically not possible to
upgrade Operator SDK's Kubernetes dependencies until these other projects have
releases that also include the Kubernetes dependency updates.

The two projects that fall into this category are:
1. [`kubernetes-sigs/controller-runtime`][controller-runtime]
2. [`helm/helm`][helm]

To mitigate these risks, the Operator SDK contributors must work with these projects
to make the necessary upstream dependency changes and to make releases containing these
changes.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

Operator SDK's existing e2e suite will be used to verify these changes. At a minimum,
the e2e suite will need to be updated to run the tests against a Kubernetes 1.17
cluster.

Since our tests make use of Kubernetes utilities and APIs, other changes may be
necessary depending on the specific changes in Kubernetes 1.17.

### Upgrade / Downgrade Strategy

To help users upgrade their projects, Operator SDK provides a migration guide that
documents the steps that operator develoeprs must take to migrate their project
to a new version of the SDK.

The migration guide for the version containing the Kubernetes 1.17 dependency
update will document the changes necessary in the project's `go.mod` file, and
it will call out any specific breaking changes that may impact operator projects
along with specific instructions for mitigating the breaking change, if applicable.

### Version Skew Strategy

With Operator SDK, Version skew is typically a concern when deploying an operator to a
Kubernetes cluster where the operator's Kubernetes client libraries are at a different
version than the Kubernetes API server with which it is interacting.

The Kubernetes API compatibility matrix documents which clients and APIs are supported by which API server versions and vice versa.

See the [`kubernetes/client-go` documentation about versioning][version-skew], which describes the supported version skew between clients and servers.

[version-skew]: https://github.com/kubernetes/client-go#versioning

## Implementation History

Major milestones in the life cycle of a proposal should be tracked in `Implementation
History`.

## Drawbacks

The only drawback is the potential for breaking changes in client code due to breaking
changes introduced by the Kubernetes 1.17 version update. However, this is unavoidable
since Kubernetes minor versions almost always contain breaking changes that impact 
controller-runtime and Operator SDK.

## Alternatives

None

## Infrastructure Needed (optional)

N/A

[operator-sdk-doc]:  ../../../doc
[controller-runtime]: https://github.com/kubernetes-sigs/controller-runtime
[helm]: https://github.com/helm/helm
