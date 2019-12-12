---
title: Neat-Enhancement-Idea
authors:
  - "@estroz"
reviewers:
  - "@joelanford"
  - "@dmesser"
approvers:
  - "@joelanford"
  - "@dmesser"
creation-date: 2019-09-12
last-updated: 2019-12-11
status: implementable
see-also:
  - "./cli-ux-phase1.md"  
---

# sdk integration with olm

## Release Signoff Checklist

- \[x] Enhancement is `implementable`
- \[x] Design details are appropriately documented from clear requirements
- \[ ] Test plan is defined
- \[ ] Graduation criteria for dev preview, tech preview, GA
- \[ ] User-facing documentation is created in openshift/docs

## Summary

The [Operator Lifecycle Manager (OLM)][olm] is a set of cluster resources that manage the lifecycle of an Operator. OLM can be installed onto a Kubernetes cluster to provide a robust Operator management system for any cluster users. The Operator SDK (SDK) should be able to interact with OLM to a degree that gives any user the ability to deploy their Operator and tear it down using OLM, all in a reproducible fashion. This proposal aims to describe integration of OLM into the SDK for deployment and teardown.

## Motivation

OLM is an incredibly useful cluster management tool. There is currently no integration between SDK and OLM that encourages running an Operator with the latter.

### Goals

#### General

- Operator developers can use `operator-sdk` to quickly deploy OLM on a given Kubernetes cluster
- Operator developers can use `operator-sdk` to run their Operator under OLM
- Operator developers can use `operator-sdk` to build a catalog/bundle containing their Operator for use with OLM

#### Specific

- `operator-sdk` creates a [bundle][bundle] from an Operator project to deploy with OLM
- `operator-sdk` has a CLI interface to interact with OLM
- `operator-sdk` installs a specific version of OLM onto Kubernetes cluster
- `operator-sdk` uninstalls a specific version of OLM onto Kubernetes cluster
- `operator-sdk` accepts a bundle and deploys that operator onto an OLM-enabled Kubernetes cluster
- `operator-sdk` accepts a bundle and removes that operator from an OLM-enabled Kubernetes cluster

### Non-Goals

- Replicate mechanisms and abilities of OLM in `operator-sdk`.

## Proposal

### User Stories

The following stories pertain to both upstream Kubernetes and OpenShift cluster types.

#### Story 1

I should be able to install a specific version of OLM onto a cluster

#### Story 2

I should be able to uninstall a specific version of OLM from a cluster

#### Story 3

I should be able to deploy a specific version of an Operator using OLM and a bundle director.

#### Story 4

I should be able to remove a specific version of an Operator deployed using `operator-sdk` via OLM from a cluster.

#### Story 5

I should be able to specify one or more [required manifests](#olm-resources) saved locally or have `operator-sdk` generate them from bundled data during deployment.

### Implementation Details/Notes/Constraints

Initial PR: https://github.com/operator-framework/operator-sdk/pull/1912

#### Use of operator-registry

The SDK's approach to deployment should be as general and reliant on existing mechanisms as possible. To that end, [`operator-registry`][registry] should be used since it defines what a bundle contains and how one is structured. `operator-registry` libraries should be used to create and serve bundles, and interact with package manifests.

The idea is to create a `Deployment` containing the latest `operator-registry` [image][registry-image] to initialize a bundle database and run a registry server serving that database using binaries contained in the image. The `Deployment` will contain volume mounts from a `ConfigMap` containing bundle files and a package manifest for an Operator. Using manifest data in the `ConfigMap` volume source, the registry initializer can build a local database and serve that database through the `Service`. OLM-specific resources created by the SDK or supplied by a user, described below, will establish communication between this registry server and OLM.

#### OLM resources

OLM understands `operator-registry` servers and served data through several objects. A [`CatalogSource`][olm-catalogsource] specifies how to communicate with a registry server. A [`Subscription`][olm-subscription] links a particular CSV channel to a `CatalogSource`, indicating from which `CatalogSource` OLM should pull an Operator. Another OLM resource that _may_ be required is an [`OperatorGroup`][olm-operatorgroup], which provides Operator namespacing information to OLM. OLM creates a globally-scoped `OperatorGroup` by default, which can be used for globally-scoped Operators.

These resources can be created from bundle data with minimal user input. They can also be created from manifests defined by the user; however, the SDK cannot make guarantees that user-defined manifests will work as expected.

#### OperatorGroups and tenancy requirements

[`OperatorGroup`][olm-operatorgroup]'s configure CSV tenancy in multiple
namespaces in a cluster. Each Operator must be a
[member][olm-operatorgroup-membership] with one `OperatorGroup` resource in
the cluster, which defines a set of namespaces the CSV can exist in. A CSV's
`installModes` determine what [type][olm-operatorgroup-installmodes] of
`OperatorGroup` it can be a member of.

No two `OperatorGroup`'s can exist in the same namespace, and a CSV with
membership in an `OperatorGroup` of a type it does not support (determined
by `installModes`) will transition to a failure state.

Given these rules and constraints, Operator developers may have a tough time
writing an `OperatorGroup` for their Operator initially. To assist them,
`operator-sdk` should automate `OperatorGroup` "compilation" if one is not
supplied.

To perform compilation, the user can optionally supply the desired install
mode type by which the CSV is installed, and the set of namespaces (may be all
namespaces, `""`) in which the CSV will be installed. The compilation
algorithm is as follows:

```
1. If an OperatorGroup manifest is supplied:
    1. Use the one supplied and return.
2. Else if an OperatorGroup manifest is not supplied, compile an OperatorGroup g:
    1. If no installMode and set of namespaces is supplied:
        1. Initialize g as type OwnNamespace by setting g's targetNamespaces to the Operator's namespace, and return.
    2. Else if an installMode and set of namespaces is supplied:
        1. Validate the set of namespaces against the install mode's constraints and the Operator's namespace.
        2. Initialize g as the desired type with the set of namespaces and return.
```

Managing `OperatorGroup` resources for multiple Operators _before_ deployment
is attempted is a more complex problem, but prevents annoying-to-debug
deployment issues that will occur in the following scenarios:

- A user wants to deploy two or more Operators with CSV install modes
incompatible for one `OperatorGroup` to handle in the name namespace.
- A user wants to create an `OperatorGroup` in a namespace that already has
an `OperatorGroup`.
    - The new and existing `OperatorGroup` namespace intersection is:
        - Equivalent to the set of new and existing namespaces (they have the
          same set).
        - The empty set (not intersecting).
        - A strict subset of either namespace set.

A solution to these types of conflicts is the following two algorithms:

Algorithm for creating an `OperatorGroup`:
```
1. Follow the compilation algorithm above to create an OperatorGroup g.
2. Determine whether an OperatorGroup exists in a given namespace n.
3. If no OperatorGroup exists in n:
    1. If h was not compiled by operator-sdk:
        1. Label g with a static label to signify g was not created by operator-sdk.
    2. Else if h was created by operator-sdk:
        1. Label g with a static label to signify g was created by operator-sdk.
    3. Create g in n and return.
4. Else if an OperatorGroup h exists in n:
    1. If h was not compiled by operator-sdk, return an error.
    2. Else if h was compiled by operator-sdk:
        1. Determine which CSV's are members of h, h's targetNamespaces hn, and g's targetNamespaces gn.
        2. If gn is equivalent to hn, return.
        3. Else if the intersection of gn and hn is the empty set or a subset of either:
            1. Label g with a static label to signify g was created by operator-sdk.
            2. Create g in another namespace m and return.
```

Algorithm for deleting an `OperatorGroup`:
```
1. Determine whether an OperatorGroup exists in a given namespace n.
2. If no OperatorGroup exists in n, return.
3. Else if an OperatorGroup g exists in n:
    1. If g is not labeled with an operator-sdk static label, return.
    2. Else if g is labeled with an operator-sdk static label:
        1. Determine the set of CSV's cs that are members of g.
        2. If cs is the empty set:
            1. Delete g and return.
        3. Else if cs is not the empty set, return.
```

Notes on these algorithms:
- Labeling allows `operator-sdk` to determine whether an `OperatorGroup` can
be deleted; `OperatorGroup`'s not compiled by `operator-sdk` should not be
deleted in any case.
- An `OperatorGroup` not compiled by `operator-sdk` is considered a user-
managed resource. All conflicts must be resolved by the user, so an error
is returned if a non-compiled `OperatorGroup` is already present in a namespace.
- Deleting an `OperatorGroup` associated with 1..N CSVs will cause those CSVs
to transition to a failure state, so we should not delete if this is the case.

[olm-operatorgroup-membership]: https://github.com/operator-framework/operator-lifecycle-manager/blob/1cb0681/doc/design/operatorgroups.md
[olm-operatorgroup-installmodes]: https://github.com/operator-framework/operator-lifecycle-manager/blob/1cb0681/doc/design/operatorgroups.md

#### Use of operator-framework/api validation

Static validation is necessary for users to determine problems before deploying their Operator. As we all know, static bugs are usually more tractable than runtime bugs, especially those discovered in a live cluster. The [`operator-framework/api`][of-api] repo intends to house a validation library for static, and potentially runtime, validation. The SDK should use this library as the source of truth for the qualities of a valid OLM manifest. This repo is a work-in-progress, and should be used as soon as it is ready.

### Risks and Mitigations

There are fewer risks with this approach than others because external libraries that define OLM components are used whenever possible, ensuring maximum compatibility.

One risk factor is how hidden OLM nuances are from users. Much of how an Operator is deployed using a registry and OLM resources like `Subscription`'s is complex, and understanding each component is necessary for true self sufficiency. However good documentation can help direct users towards solutions. There is also an ongoing effort to reduce the complexity of OLM metadata requirements.

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

[olm]:https://github.com/operator-framework/operator-lifecycle-manager/
[olm-operatorgroup]:https://github.com/operator-framework/operator-lifecycle-manager/blob/1cb0681/doc/design/operatorgroups.md
[olm-subscription]:https://github.com/operator-framework/community-operators/blob/master/docs/testing-operators.md#7-create-a-subscription
[olm-catalogsource]:https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/philosophy.md#catalogsource
[registry]:https://github.com/operator-framework/operator-registry/
[bundle]:https://github.com/operator-framework/operator-registry/#manifest-format
[registry-image]:https://quay.io/organization/openshift/origin-operator-registry
[of-api]:https://github.com/operator-framework/api
