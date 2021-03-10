---
title: Release Guide
linkTitle: Releasing
weight: 30
---

These steps describe how to conduct a release of the operator-sdk repo using example versions.
Replace these versions with the current and new version you are releasing, respectively.

Table of contents:

- [Major and minor releases](#major-and-minor-releases)
- [Patch releases](#patch-releases)
- [`scorecard-test-kuttl` image releases](#scorecard-test-kuttl-image-releases)
- [Release tips](#helpful-tips-and-information)

## Prerequisites

- [`git`](https://git-scm.com/downloads)
- [`make`](https://www.gnu.org/software/make/)
- [`sed`](https://www.gnu.org/software/sed/)

##### MacOS users

Install GNU `sed` and `make` which may not be by default:

```sh
brew install gnu-sed make
```

## Major and Minor releases

We will use the `v1.3.0` release version in this example.

### Before starting

1. A release branch must be created and [mapped][netlify-deploy] _before the release begins_
to appease the Netlify website configuration demons. You can ping SDK [approvers][doc-owners] to ensure a
[release branch](#release-branches) is created prior to the release and that this mapping is created.
If you have the proper permissions, you can do this by running the following,
assuming the upstream SDK is the `upstream` remote repo:
  ```sh
  git checkout master
  git pull
  git checkout -b v1.3.x
  git push -u upstream v1.3.x
  ```
1. Create and merge a commit that updates the top-level [Makefile] variable `IMAGE_VERSION`
to the upcoming release tag `v1.3.0`. This variable ensures sample projects have been tagged
correctly prior to the release commit.
  ```sh
  sed -i -E 's/(IMAGE_VERSION = ).+/\1v1\.3\.0/g' Makefile
  ```
1. Lock down the `master` branch to prevent further commits before the release completes:
  1. Go to `Settings -> Branches` in the SDK repo.
  1. Under `Branch protection rules`, click `Edit` on the `master` branch rule.
  1. In section `Protect matching branches` of the `Rule settings` box, increase the number of required approving reviewers to 6.

### 1. Create and push a release commit

Create a new branch to push the release commit:

```sh
export RELEASE_VERSION=v1.3.0
git checkout master
git pull
git checkout -b release-$RELEASE_VERSION
```

Run the pre-release `make` target:

```sh
make prerelease
```

The following changes should be present:

- `changelog/generated/v1.3.0.md`: commit changes (created by changelog generation).
- `changelog/fragments/*`: commit deleted fragment files (deleted by changelog generation).
- `website/content/en/docs/upgrading-sdk-version/v1.3.0.md`: commit changes (created by changelog generation).
- `website/config.toml`: commit changes (modified by release script).

Commit these changes and push:

```sh
git add --all
git commit -m "Release $RELEASE_VERSION"
git push -u origin release-$RELEASE_VERSION
```

### 2. Create and merge a new PR

Create and merge a new PR for the commit created in step 1. You can force-merge your PR to the locked-down `master`
if you have admin access to the operator-sdk repo, or ask an administrator to do so.

### 3. Unlock the `master` branch

Unlock the branch by changing the number of required approving reviewers in the `master` branch rule back to 1.

### 4. Create and push a release tag

```sh
make tag
git push upstream refs/tags/$RELEASE_VERSION
```

### 5. Fast-forward the `latest` and release branches

The `latest` branch points to the latest release tag to keep the main website subdomain up-to-date.
Run the following commands to do so:

```sh
git checkout latest
git reset --hard tags/$RELEASE_VERSION
git push -f upstream latest
```

Similarly, to update the release branch, run:

```sh
git checkout v1.3.x
git reset --hard tags/$RELEASE_VERSION
git push -f upstream v1.3.x
```

### 6. Post release steps

- Make an [operator-framework Google Group][of-ggroup] post.
- Post to Kubernetes slack in #kubernetes-operators and #operator-sdk-dev.
- In the [GitHub milestone][gh-milestones], bump any open issues to the following release.


## Patch releases

We will use the `v1.3.1` release version in this example.

### Before starting

1. Create and merge a commit that updates the top-level [Makefile] variable `IMAGE_VERSION`
to the upcoming release tag `v1.3.1`. This variable ensures sample projects have been tagged
correctly prior to the release commit.
  ```sh
  sed -i -E 's/(IMAGE_VERSION = ).+/\1v1\.3\.1/g' Makefile
  ```
1. Lock down the `v1.3.x` branch to prevent further commits before the release completes:
  1. Go to `Settings -> Branches` in the SDK repo.
  1. Under `Branch protection rules`, click `Edit` on the `v.*` branch rule.
  1. In section `Protect matching branches` of the `Rule settings` box, increase the number of required approving reviewers to 6.

### 1. Create and push a release commit

Create a new branch from the release branch, which should already exist for the desired minor version,
to push the release commit to:

```sh
export RELEASE_VERSION=v1.3.1
git checkout v1.3.x
git pull
git checkout -b release-$RELEASE_VERSION
```

Run the pre-release `make` target:

```sh
make prerelease
```

The following changes should be present:

- `changelog/generated/v1.3.0.md`: commit changes (created by changelog generation).
- `changelog/fragments/*`: commit deleted fragment files (deleted by changelog generation).

Commit these changes and push:

```sh
git add --all
git commit -m "Release $RELEASE_VERSION"
git push -u origin release-$RELEASE_VERSION
```

### 2. Create and merge a new PR

Create and merge a new PR for the commit created in step 1. You can force-merge your PR to the locked-down `v1.3.x`
if you have admin access to the operator-sdk repo, or ask an administrator to do so.

### 3. Unlock the `v1.3.x` branch

Unlock the branch by changing the number of required approving reviewers in the `v.*` branch rule back to 1.

### 4. Create and push a release tag

```sh
make tag
git push upstream refs/tags/$RELEASE_VERSION
```

### 5. Fast-forward the `latest` branch

The `latest` branch points to the latest release tag to keep the main website subdomain up-to-date.
Run the following commands to do so:

```sh
git checkout latest
git reset --hard tags/$RELEASE_VERSION
git push -f upstream latest
```

### 6. Post release steps

- Make an [operator-framework Google Group][of-ggroup] post.
- Post to Kubernetes slack in #kubernetes-operators and #operator-sdk-dev.
- In the [GitHub milestone][gh-milestones], bump any open issues to the following release.

**Note**
In case there are non-transient errors while building the release job, you must:
1. Revert the release PR. To do so, create a PR which reverts step [2](#2-create-and-merge-a-new-pr).
2. Fix what broke in the release branch.
3. Re-run the release with an incremented minor version to avoid Go module errors (ex. if v1.3.1 broke, then re-run the release as v1.3.2). Patch versions are cheap so this is not a big deal.

## `scorecard-test-kuttl` image releases

The `quay.io/operator-framework/scorecard-test-kuttl` image is released separately from other images because it
contains the [`kudobuilder/kuttl`](https://hub.docker.com/r/kudobuilder/kuttl/tags) image, which is subject to breaking changes.

Release tags of this image are of the form: `scorecard-kuttl/vX.Y.Z`, where `X.Y.Z` is _not_ the current operator-sdk version.
For the latest version, query the [operator-sdk repo tags](https://github.com/operator-framework/operator-sdk/tags) for `scorecard-kuttl/v`.

The only step required is to create and push a tag.
This example uses version `v2.0.0`, the first independent release version of this image:

```sh
export RELEASE_VERSION=scorecard-kuttl/v2.0.0
make tag
git push upstream refs/tags/$RELEASE_VERSION
```

The [`deploy/image-scorecard-test-kuttl`](https://github.com/operator-framework/operator-sdk/actions/workflows/deploy.yml)
Action workflow will build and push this image.


## Helpful tips and information

### Binaries and signatures

Binaries will be signed using our CI system's GPG key. Both binary and signature will be uploaded to the release.

### Release branches

Each minor release has a corresponding release branch of the form `vX.Y.x`, where `X` and `Y` are the major and minor
release version numbers and the `x` is literal. This branch accepts bug fixes according to our [backport policy][backports].

##### Cherry-picking

Once a minor release is complete, bug fixes can be merged into the release branch for the next patch release.
Fixes can be added automatically by posting a `/cherry-pick v1.3.x` comment in the `master` PR, or manually by running:

```sh
git checkout v1.3.x
git checkout -b cherrypick/some-bug
git cherry-pick <commit>
git push upstream cherrypick/some-bug
```

Create and merge a PR from your branch to `v1.3.x`.

### GitHub release information

GitHub releases live under the [`Releases` tab][release-page] in the operator-sdk repo.


[netlify-deploy]:https://docs.netlify.com/site-deploys/overview/#deploy-summary
[doc-owners]: https://github.com/operator-framework/operator-sdk/blob/master/OWNERS
[release-page]:https://github.com/operator-framework/operator-sdk/releases
[backports]:/docs/upgrading-sdk-version/backport-policy
[of-ggroup]:https://groups.google.com/g/operator-framework
[gh-milestones]:https://github.com/operator-framework/operator-sdk/milestones
[Makefile]:https://github.com/operator-framework/operator-sdk/blob/master/Makefile
