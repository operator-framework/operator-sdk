---
title: "Operator Best Practices"
linkTitle: "Best practices"
weight: 1
description: This guide describes the best practices concepts to write operators.
---

## Development

Considerations for Operator developers:

- An Operator should manage a single type of application, essentially following the UNIX principle: do one thing and do it well.

- If an application consists of multiple tiers or components, multiple Operators should be written one for each of them. For example, if the application consists of Redis, AMQ, and MySQL, there should be 3 Operators, not one.

- If there is significant orchestration and sequencing involved, an Operator should be written that represents the entire stack, in turn delegating to other Operators for orchestrating their part of it.

- Operators should own a CRD and only one Operator should control a CRD on a cluster. Two Operators managing the same CRD is not a recommended best practice. An API that exists with multiple implementations is a typical example of a no-op Operator. The no-op Operator doesn't have any deployment or reconciliation loop to define the shared API and other Operators depend on this Operator to provide one implementation of the API, e.g. similar to PVCs or Ingress. 

- Inside an Operator, multiple controllers should be used if multiple CRDs are managed. This helps in separation of concerns and code readability. Note that this doesn't necessarily mean that we need to have one container image per controller, but rather one reconciliation loop (which could be running as part of the same Operator binary) per CRD.

- An Operator shouldn't deploy or manage other operators (such patterns are known as meta or super operators or include CRDs in its Operands). It's the Operator Lifecycle Manager's job to manage the deployment and lifecycle of operators. For further information check [Dependency Resolution][Dependency Resolution].

- If multiple operators are packaged and shipped as a single entity by the same CSV for example, then it is recommended to add all owned and required CRDs, as well as all deployments for operators that manage the owned CRDs, to the same CSV.

- Writing an Operator involves using the Kubernetes API, which in most scenarios will be built using same boilerplate code. Use a framework like the Operator SDK to save yourself time with this and to also get a suite of tooling to ease development and testing.

- Operators shouldn’t make any assumptions about the namespace they are deployed in and they should not use hard-coded names of resources that they expect to already exist.

- Operators shouldn’t hard code the namespaces they are watching. This should be configurable - having no namespace supplied is interpreted as watching all namespaces

- Semantic versioning (aka semver) should be used to version an Operator. Operators are long-running workloads on the cluster and its APIs are potentially in need of support over a longer period of time. Use the [semver.org guidelines](https://semver.org) to help determine when and how to bump versions when there are breaking or non-breaking changes.

- Kubernetes API versioning guidelines should be used to version Operator CRDs. Use the [Kubernetes sig-architecture guidelines](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#so-you-want-to-change-the-api) to get best practices on when to bump versions and when breaking changes are acceptable.

- When defining CRDs, you should use OpenAPI spec to create a structural schema for your CRDs.

- Operators are instrumented to provide useful, actionable metrics to external systems (e.g. monitoring/alerting platforms).  Minimally, metrics should represent the software's health and key performance indicators, as well as support the creation of [service levels indicators](https://en.wikipedia.org/wiki/Service_level_indicator) such as throughput, latency, availability, errors, capacity, etc.

- Operators may create objects as part of their operational duty. Object accumulation can consume unnecessary resources, slow down the API and clutter the user interface. As such it is important for operators to keep good hygiene and to clean up resources when they are not needed. Here are instructions on [how to handle cleanup on deletion][advanced-topics].

### Summary

- One Operator per managed application
- Multiple operators should be used for complex, multi-tier application stacks
- CRD can only be owned by a single Operator, shared CRDs should be owned by a separate Operator
- One controller per custom resource definition
- Use a tool like Operator SDK
- Do not hard-code namespaces or resources names
- Make watch namespace configurable
- Use semver / observe Kubernetes guidelines on versioning APIs
- Use OpenAPI spec with structural schema on CRDs
- Operators expose metrics to external systems
- Operators cleanup resources on deletion

## Running On-Cluster

Considerations for on-cluster behavior

- Like all containers on Kubernetes, Operators need not run as root unless absolutely necessary. Operators should come with their own ServiceAccount and not rely on the `default`.

- Operators should not self-register their CRDs. These are global resources and careful consideration needs to be taken when setting those up. Also this requires the Operator to have global privileges which is potentially dangerous compared to that little extra convenience.

- Operators use CRs as the primary interface to the cluster user. As such, at all times, meaningful status information should be written to those objects unless they are solely used to store data in a structured schema.

- Operators should be updated frequently according to server versioning.

- Operators need to support updating managed applications (Operands) that were set up by an older version of the Operator. There are multiple models for this:

| Model | Description | 
| ------ | ----- |
| **Operator fan-out** | where the Operator allows the user to specify the version in the custom resource |
| **single version** | where the Operator is tied to the version of the operand. |
| **hybrid approach** | where the Operator is tied to a range of versions, and the user can select some level of the version. |

- An Operator should not deploy another Operator - an additional component on cluster should take care of this (OLM).

- When Operators change their APIs, CRD conversion (webhooks) should be used to deal with potentially older instances of them using the previous API version.

- Operators should make it easy for users to use their APIs - validating and rejecting malformed requests via extensive Open API validation schema on CRDs or via an admission webhook is good practice.

- The Operator itself should be really modest in its requirements - it should always be able to deploy by deploying its controllers, no user input should be required to start up the Operator.

- If user input is required to change the configuration of the Operator itself, a Configuration CRD should be used. Init-containers as part of the Operator deployments can be used to create a default instance of those CRs and then the Operator manages their lifecycle.

### Summary:

On the cluster, an Operator...

- Does not run as root
- Does not self-register CRDs
- Does not install other Operators
- Does rely on dependencies via package manager (OLM)
- Writes meaningful status information on Custom Resources objects unless pure data structure
- Should be capable of updating from a previous version of the Operator
- Should be capable of managing an Operand from an older Operator version
- Uses CRD conversion (webhooks) if API/CRDs change
- Uses OpenAPI validation / Admission Webhooks to reject invalid CRs
- Should always be able to deploy and come up without user input
- Offers (pre)configuration via a `“Configuration CR”` instantiated by InitContainers

[Dependency Resolution]: https://olm.operatorframework.io/docs/concepts/olm-architecture/dependency-resolution/
[advanced-topics]:/docs/building-operators/golang/advanced-topics#handle-cleanup-on-deletion
