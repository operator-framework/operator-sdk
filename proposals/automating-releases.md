---
title: automating-releases
authors:
  - @estroz
reviewers:
  - @jmrodri
  - @joelanford
  - @theishah
approvers:
  - TBD
creation-date: 2020-09-19
last-updated: 2020-09-24
status: implementable
---

# automating-releases

## Release Signoff Checklist

- \[ \] Enhancement is `implementable`
- \[ \] Design details are appropriately documented from clear requirements
- \[ \] Test plan is defined
- \[ \] Graduation criteria for dev preview, tech preview, GA
- \[ \] User-facing documentation is created [for the website][operator-sdk-doc]

## Open Questions

1. We currently build images for an architecture on machines of that architecture.
This is not a requirement for Go binaries themselves, but might be for Ansible/Helm
dependencies since they install binaries for particular architectures. What are
these dependencies and can they be installed in a container during image build?
1. This proposal encompasses all binaries/images currently built for an operator-sdk release.
Once Ansible- and Helm-related code are split into their respective repos, will
this release process follow?
1. Continue using our [changelog generator][sdk-changelog-generator], or use goreleaser's
[generator][goreleaser-changelog]?

## Summary

This proposal outlines an automated release pipeline for the operator-sdk repository
that includes both binary and image builds. The [`goreleaser`][goreleaser] release tool
will replace the existing combination of the release script `release.sh`, manual `git`
steps, and manual release publishing with a one "click" solution. To users, no changes
will occur since the release artifacts are not changing. To developers, the release
process will become much simpler, hopefully to the point where a new contributor can
run the release with little process knowledge.

## Motivation

Releasing the operator-sdk repo currently requires several steps that are mostly manual,
with some automation around building binaries for all architectures supported. Tools like
[`goreleaser`][goreleaser] can automate this process entirely by tagging code, and building
and publishing binaries and images with one "click".

## Goals

- Remove all manual steps from the release process
- Centralize release configuration
- One-click releasing

### Non-Goals

- Change the content or constituency of output release artifacts in any way
- Change the testing process in any way
- Change our CI provider

## Proposal

### User Stories

#### Story 1 - Release the operator-sdk repo in one click

As a contributor I want to run the release in as few operations as possible.
This feature will enable new contributors to release the repo easily, instead
of relying on a long onboarding document and the expertise of a few contributors.

#### Story 2 - Easily modify release configuration

As a contributor I want to easily improve the release configuration whenever necessary,
which includes adding new release architectures. Configuration should be documented,
comprehensible, and centralized.

### Implementation Details/Notes/Constraints

The [`goreleaser`][goreleaser] release tool satisfies the above goals and user stories:
- Manages code tags
- Builds and pushes [binaries][goreleaser-build] and [images][goreleaser-docker]
  - Applies to `operator-sdk`, `ansible-operator`, and `helm-operator`
- Centralized configuration and a well-documented [configuration spec][goreleaser-config]
- Handles the full release process with one click

`goreleaser` configuration is CI-agnostic, so if we choose to switch CI systems in the future
we can bring our configuration with us verbatim.

Additionally it supports generating release notes based on commits, or from a
[custom source][goreleaser-custom-changelog]; we currently use a custom changelog generator
which we can continue using.

### Risks and Mitigations

- Contributors need to learn a new configuration format. Luckily the goreleaser
[config file format][goreleaser-config] is straightforward and has great documentation.
- Release tool development is controlled by another organization, which can lead to undesirable bugs/changes.
This is a manageable risk because the project is open source, to which we can contribute.

## Design Details

### Test Plan

1. Create a goreleaser configuration that creates the exact same release artifacts the current process does.
1. Push configuration to a test repository.
1. Run a test release.
1. Ensure:
  1. Tag is present remotely.
  1. Release notes/binaries are published as expected.
  1. Images are pullable and run on the desired architectures.

### Graduation Criteria

Once test plan has been executed and passed, the goreleaser configuration can be pushed to the
operator-sdk repo and unused release scripts/code removed.

#### Examples

TBD

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

N/A

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

The `goreleaser` binary will be updated whenever a new release containing a bug fix we want is available.

## Implementation History

Current release [script][sdk-release-script] and [documentation][sdk-release-doc].

## Drawbacks

See [risks and mitigations](#risks-and-mitigations).

## Alternatives

- Combining native Github actions like [create-release](https://github.com/actions/create-release).
  - Still requires glue code and multiple actions have different configs that are not documented as well.
- A traditional build system like [Bazel][bazel]. While this could work, it is quite heavy-handed for
what we need and would likely involve writing glue code.

## Infrastructure Needed (optional)

None, this can continue using our current CI infrastructure.


[operator-sdk-doc]:https://sdk.operatorframework.io/
[sdk-changelog-generator]:https://github.com/operator-framework/operator-sdk/tree/v1.0.1/hack/generate/changelog
[sdk-release-script]:https://github.com/operator-framework/operator-sdk/tree/v1.0.1/release.sh
[sdk-release-doc]:https://v1-0-x.sdk.operatorframework.io/docs/contribution-guidelines/release/
[goreleaser]:https://goreleaser.com/customization/
[goreleaser-config]:https://goreleaser.com/customization/
[goreleaser-build]:https://goreleaser.com/customization/build/
[goreleaser-docker]:https://goreleaser.com/customization/docker/
[goreleaser-changelog]:https://goreleaser.com/customization/release/#customize-the-changelog
[goreleaser-custom-changelog]:https://goreleaser.com/customization/release/#custom-release-notes
[bazel]:https://docs.bazel.build/versions/3.5.0/bazel-overview.html
