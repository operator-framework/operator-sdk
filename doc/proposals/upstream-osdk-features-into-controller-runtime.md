---
title: upstream-OSDK-features-to-controller-runtime
authors:
  - "@hasbro17"
reviewers:
  - TBD
  - "@joelanford"
  - "@dmesser"
approvers:
  - TBD
  - "@joelanford"
  - "@dmesser"
creation-date: 2019-09-22
last-updated: 2019-09-22
status: implementable
---

# upstream-OSDK-features-to-controller-runtime

## Release Signoff Checklist

- Enhancement is `implementable`
- Design details are appropriately documented from clear requirements
- Test plan is defined


## Summary

An operator project scaffolded with the Operator SDK will primarily use library code that lives upstream in the controller-runtime project. There is a small set of packages in the SDK that extends the functionality of the controller-runtime APIs to cover more specific use cases. Most of these features along with their documentation should be contributed upstream to the controller-runtime.

Additionally the Operator SDK is preparing to use Kubebuilder for scaffolding Go operator projects for better alignment with the upstream community. See [kubebuilder-project-board][kubebuilder-project-board] and [upstream proposal for the integration of Operator SDK and Kubebuilder][sdk-kubebuilder-integration-proposal]. As part of this integration some features from the SDK’s scaffolding machinery need to be contributed upstream to Kubebuilder before the SDK can use Kubebuilder as its upstream.


## Motivation

Contributing the extra features in the SDK to the controller-runtime would make them available to all users of the controller-runtime. This also reduces the maintenance burden in the SDK by since there are more contributors and users upstream who can improve on these features over time.

Contributing scaffolding enhancements to Kubebuilder that cover the SDK’s use cases removes blockers in the SDK’s proposal to use Kubebuilder as upstream, and helps align the SDK and Kubebuilder on a common project layout and workflow.


### Goals

- The packages and features that are suitable for upstream contribution should be made available in the controller-runtime such that they cover the same use cases.
- Downstream SDK users can easily switch over to using those features in the controller-runtime. In most cases this should amount to just changing the import paths. For function signature changes there should be documentation to explain the breaking changes.  
- The contributed features should have sufficient documentation in the upstream godocs.
- Once available upstream, those packages and APIs should be marked as deprecated and eventually removed from the SDK.


### Non-Goals

- Not all SDK library code is suitable for upstream contributions.
  - For instance the test-framework library is closely tied to the SDK’s testing workflow and not generally applicable outside SDK projects.
  - The SDK's leader-for-life leader election package already has an alternative in the controller-runtime’s leader election package which uses lease based leader election.


## Proposal

The user stories outline the individual features that are suitable for upstream contribution. Some of these features may already be merged upstream or are currently under review.


### User Stories

#### Story 1 - Dynamic RestMapper that can reload to update discovery information for new resource types

The default RestMapper used in the controller-runtime will not update to reflect new resource types registered with the API server after the RestMapper is first initialized at startup. See [operator-sdk #1328][operator-sdk-1328] and [controller-runtime #321][controller-runtime-321].

The SDK has pkg/restmapper (see [operator-sdk #1329][operator-sdk-1329]) that provides a dynamic RestMapper which will reload the cached rest mappings on lookup errors due to a stale cache. This dynamic restmapper is currently under review for upstream contribution with some improvements like thread safety and rate limiting. See [controller-runtime #554][controller-runtime-554].


#### Story 2 - GenerationChangedPredicate that can filter watch events with no generation change

The SDK provides a predicate called GenerationChangedPredicate in [pkg/predicate][sdk-pkg-predicate] that will filter out update events for objects that have no change in their metadata.Generation. This is commonly used for ignoring update events for CustomResource objects that only have their status block updated with no change to the spec block.

This feature has already been incorporated upstream with godocs on the predicate and its caveats. See [controller-runtime-553][controller-runtime-553] and [GenerationChangedPredicate godocs][gen-change-predicate-godocs].


#### Story 3 - Add command line flags to make the controller-runtime’s zap based logger configurable

The SDK has [pkg/log/zap][sdk-pkg-log-zap] that provides a zap based logr logger that allows a number of fine grained configurations (e.g debug level, encoder formatting) via [command line flags][sdk-zap-cmd-flags] passed to the operator.

The controller-runtime’s zap based logger has recently been made configurable via functional options that should allow all the configurations in the SDK’s own zap logger (see [controller-runtime #560][controller-runtime-560]). This needs to be followed up by adding predefined options for each configuration of the logger to the controller-runtime so that users don’t have to write it themselves.

Ideally the flagset for setting all the logger configurations could also live upstream but given that the controller-runtime’s [pkg/log/zap][cr-pkg-log-zap] allows instantiating multiple zap loggers with different configs, it may not be suitable to have a global flagset that provides a singular configuration for all instantiated loggers.
This point needs more discussion and it’s possible that the configuration flags may have to live downstream in the SDK.


#### Story 4 - Operator Developers can use SDK’s method of building images that run as non-root users in Kubebuilder projects

Kubebuilder scaffolded projects would previously run the operator base images with the user as root. See Kubebuilder’s [pkg/scaffold/v2/dockerfile.go][kb-scaffold-dockerfile].

Operator SDK scaffolds a project Dockerfile such that it runs as non-root by default and allows the operator image to run with arbitrary UIDs on openshift. For details, see the [openshift container guidelines][openshift-container-guidelines] for non-root images, and how the SDK includes the user setup in the image build and uses a custom entrypoint:

- [internal/pkg/scaffold/build_dockerfile.go][sdk-scaffold-dockerfile]
- [internal/pkg/scaffold/entrypoint.go][sdk-scaffold-entrypoint]
- [internal/pkg/scaffold/usersetup.go][sdk-scaffold-user-setup]

Kubebuilder should support scaffolding projects that will allow the base image to run as non-root and support arbitrary user ids. 
Currently with [kubebuilder #983][kubebuilder-983], a non-root base image should be supported by Kubebuilder.


#### Story 5 - Operator Developers can use the prometheus-operator’s ServiceMonitor API to configure prometheus to scrape their operator metrics in Kubebuilder projects

The Operator SDK’s [pkg/metrics][sdk-pkg-metrics] has helpers that let’s operators configure and create [ServiceMonitors][service-monitor-doc] that lets a prometheus instance on a cluster target the operator’s Service object that exposes operator metrics.

Instead of having this functionality live in controller-runtime as helpers that can be called to setup ServiceMonitors, this can added as manifests that are scaffolded by Kubebuilder for an operator project. This would be similar to other resources that need to be created alongside the operator and can be customized in the manifest. Upstream issue at [kubebuilder #887][kubebuilder-887].


#### Story 6 - Operator Developers have documentation that demonstrates how to create and expose custom operator metrics

Once Kubebuilder supports scaffolding ServiceMonitor manifests, the [Kubebuilder book][kubebuilder-book] documentation on [recording custom metrics][recording-custom-metrics] should be extended to show how to expose these metrics via the ServiceMonitor.



### Risks and Mitigations

Almost all of the upstream work entails breaking changes for SDK users.
Before the features are removed from the SDK they should first be deprecated in prior release with sufficient documentation that explains how to use the equivalent features upstream in the controller-runtime.

The features that have replacements in kubebuilder will need to wait until the SDK is ready to upstream kubebuilder for Go operators before being removed.


### Test Plan

All library features being upstreamed into the controller-runtime would need to have unit tests to run as part of its CI.

Similarly any scaffolding and manifest changes to Kubebuilder would need e2e tests that verify the effects of those manifest changes as part of Kubebuilder’s CI.


[kubebuilder-project-board]: https://github.com/kubernetes-sigs/kubebuilder/projects/7
[sdk-kubebuilder-integration-proposal]: https://github.com/kubernetes-sigs/kubebuilder/blob/992ecdfd3f47e4cca79937a4fd46a0ee10f477d7/designs/integrating-kubebuilder-and-osdk.md
[operator-sdk-1328]: https://github.com/operator-framework/operator-sdk/issues/1328
[operator-sdk-1329]: https://github.com/operator-framework/operator-sdk/pull/1329
[controller-runtime-321]: https://github.com/kubernetes-sigs/controller-runtime/issues/321
[controller-runtime-554]: https://github.com/kubernetes-sigs/controller-runtime/pull/554
[sdk-pkg-predicate]: https://github.com/operator-framework/operator-sdk/blob/947a464dbe968b8af147049e76e40f787ccb0847/pkg/predicate/predicate.go
[controller-runtime-553]: https://github.com/kubernetes-sigs/controller-runtime/pull/553
[gen-change-predicate-godocs]: https://godoc.org/github.com/kubernetes-sigs/controller-runtime/pkg/predicate#GenerationChangedPredicate
[sdk-pkg-log-zap]: https://github.com/operator-framework/operator-sdk/tree/947a464dbe968b8af147049e76e40f787ccb0847/pkg/log/zap
[sdk-zap-cmd-flags]: https://github.com/operator-framework/operator-sdk/blob/947a464dbe968b8af147049e76e40f787ccb0847/pkg/log/zap/flags.go#L41-L45
[controller-runtime-560]: https://github.com/kubernetes-sigs/controller-runtime/pull/560
[cr-pkg-log-zap]: https://github.com/kubernetes-sigs/controller-runtime/blob/e825f3aafdb522bbdac626387f3c9e7d489e35a7/pkg/log/zap/zap.go#L35-L37
[kb-scaffold-dockerfile]: https://github.com/kubernetes-sigs/kubebuilder/blob/1f4fc57416ddc74ea52feb13494eb4d003d7db08/pkg/scaffold/v2/dockerfile.go#L58-L60
[openshift-container-guidelines]: https://access.redhat.com/documentation/en-us/openshift_container_platform/4.1/html/images/creating_images#images-create-guide-openshift_create-images
[sdk-scaffold-dockerfile]: https://github.com/operator-framework/operator-sdk/blob/c084b570a6af7674fd102f4ebfd3303c705e1d94/internal/pkg/scaffold/build_dockerfile.go
[sdk-scaffold-entrypoint]: https://github.com/operator-framework/operator-sdk/blob/c084b570a6af7674fd102f4ebfd3303c705e1d94/internal/pkg/scaffold/entrypoint.go
[sdk-scaffold-user-setup]: https://github.com/operator-framework/operator-sdk/blob/c084b570a6af7674fd102f4ebfd3303c705e1d94/internal/pkg/scaffold/usersetup.go
[kubebuilder-983]: https://github.com/kubernetes-sigs/kubebuilder/pull/983
[sdk-pkg-metrics]: https://github.com/operator-framework/operator-sdk/blob/f5d20c4819b98a60ec782a9a5cac784b55ea2951/pkg/metrics/service-monitor.go
[service-monitor-doc]: https://github.com/coreos/prometheus-operator/blob/master/Documentation/user-guides/getting-started.md#related-resources
[kubebuilder-887]: https://github.com/kubernetes-sigs/kubebuilder/issues/887
[kubebuilder-book]: https://book.kubebuilder.io/quick-start.html
[recording-custom-metrics]: https://book.kubebuilder.io/reference/metrics.html#publishing-additional-metrics
