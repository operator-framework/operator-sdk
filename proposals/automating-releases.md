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
last-updated: 2020-10-01
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

1. [RESOLVED] We currently build images for an architecture on machines of that architecture.
This is not a requirement for Go binaries themselves, but might be for Ansible/Helm
dependencies since they install binaries for particular architectures. What are
these dependencies and can they be installed in a container during image build?
1. This proposal encompasses all binaries/images currently built for an operator-sdk release.
Once Ansible- and Helm-related code are split into their respective repos, will
this release process follow?
1. [RESOLVED] Continue using our [changelog generator][sdk-changelog-generator], or use goreleaser's
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
- Release from semver code tags
- Encapsulates most release process complexity in one command
  - Builds [binaries][goreleaser-builds]
    - Applies to `operator-sdk`, `ansible-operator`, and `helm-operator`
  - Publishes GitHub releases
- Centralized configuration and a well-documented [configuration spec][goreleaser-config]

`goreleaser` configuration is CI-agnostic, so if we choose to switch CI systems in the future
we can bring our configuration with us verbatim.

Additionally it supports generating release notes from a [custom source][goreleaser-custom-changelog];
we currently use a custom changelog generator which we can continue using.

#### Release steps

These new release steps delegate releasing completely to CI infrastructure.

1. Commit to master:
  - Regenerated `CHANGELOG.md`
  - Removed fragments in `changelog/`
  - New migration guide `website/content/en/docs/upgrading-sdk-version/vX.Y.Z.md`
  - Updated version in `website/content/en/docs/installation/install-operator-sdk.md`
1. `git tag vX.Y.Z && git push --tags`
    1. CI runs tests on tag
    1. CI runs `make release TAG=<tag>`

Currently the `make release` only builds binaries; this new rule will set up and run `goreleaser` to:
- Build multi-arch binaries and images
- Publish a Github release with binaries and signed SHA256 hashes for each binary
  - We'll need to create a PGP key for CI
- Push images to the remote registry.

##### Binary builds

Building `operator-sdk`, `ansible-operator`, and `helm-operator` binaries is straightforward because they
can be cross-compiled locally under their own [`builds`][goreleaser-builds] task. These binaries will
be published as they are now.

##### Image builds

`goreleaser` does not yet have support for multi-arch image builds. Instead, the release config will have a
post-build hook for each task in `builds` that runs an image build script leveraging [`docker buildx`][docker-buildx],
a Docker plugin that can build an image for each arch of a multi-arch manifest list in one command.
`buildx` should be available in almost every CI environment by default as it is built into
the `docker` CLI and server v19.03+.

This removes the need for our CI to create and push manifest lists from images built on machines of the target arch.

###### Multi-stage builds

Currently our Dockerfiles require binaries be built externally with a specific name.
We should instead be encapsulating the full image build process in a multi-stage build,
in which our Dockerfiles are set up such that they first build the desired binary in a builder image (`golang:alpine`)
then copy that binary into the final image (`ubi8/ubi-minimal:latest`).

##### Publishing releases

All `goreleaser` requires to publish a release is a [Github access token][github-token] with the `repo` privilege,
and a quay.io push-enabled token (Travis CI already has one).

### Risks and Mitigations

- Contributors need to learn a new configuration format. Luckily the goreleaser
[config file format][goreleaser-config] is straightforward and has great documentation.
- Release tool development is controlled by another organization, which can lead to undesirable bugs/changes.
This is a manageable risk because the project is open source, to which we can contribute.
- [`docker buildx`][docker-buildx] is experimental and may not be easy to set up in CI.

## Design Details

### Test Plan

1. Create a goreleaser configuration that creates the exact same release artifacts the current process does.
1. Push configuration to a test repository.
1. Run a [test release][goreleaser-dry-run].
1. Ensure:
    1. Tag is present remotely.
    1. Release notes/binaries are published as expected.
    1. Images are pullable and run on the desired architectures.

### Graduation Criteria

Once test plan has been executed and passed, the goreleaser configuration can be pushed to the
operator-sdk repo and unused release scripts/code removed.

#### Examples

N/A

##### Dev Preview -> Tech Preview

N/A

##### Tech Preview -> GA

N/A

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
[goreleaser-builds]:https://goreleaser.com/customization/build/
[goreleaser-docker]:https://goreleaser.com/customization/docker/
[goreleaser-changelog]:https://goreleaser.com/customization/release/#customize-the-changelog
[goreleaser-custom-changelog]:https://goreleaser.com/customization/release/#custom-release-notes
[goreleaser-dry-run]:https://goreleaser.com/quick-start/#dry-run
[bazel]:https://docs.bazel.build/versions/3.5.0/bazel-overview.html
[github-token]:https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/creating-a-personal-access-token
[docker-buildx]:https://github.com/docker/buildx#building-multi-platform-images
