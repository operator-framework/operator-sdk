---
title: Release Guide
linkTitle: Releasing
weight: 30
---

These steps describe how to conduct a release of the operator-sdk repo using example versions.
Replace these versions with the current and new version you are releasing, respectively.

## Prerequisites

- [`git`][git]
- [`gpg`][gpg] v2.0+ and a [GPG key][gpg-key-create].
- Your GPG key is publicly available in a [public key server][gpg-upload], like https://keyserver.ubuntu.com/.

##### MacOS users

Install GNU `sed`, `make`, and `gpg` which may not be by default:

```sh
brew install gnu-sed make gnupg
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
git checkout master
git pull
git checkout -b release-v1.3.0
```

Run the pre-release `make` target:

```sh
make prerelease RELEASE_VERSION=v1.3.0
```

The following changes should be present:

- `changelog/generated/v1.3.0.md`: commit changes (created by changelog generation).
- `changelog/fragments/*`: commit deleted fragment files (deleted by changelog generation).
- `website/content/en/docs/upgrading-sdk-version/v1.3.0.md`: commit changes (created by changelog generation).
- `website/config.toml`: commit changes (modified by release script).

Commit these changes and push:

```sh
git add --all
git commit -m "Release v1.3.0"
git push -u origin release-v1.3.0
```

### 2. Create and merge a new PR

Create and merge a new PR for the commit created in step 1. You can force-merge your PR to the locked-down `master`
if you have admin access to the operator-sdk repo, or ask an administrator to do so.

### 3. Unlock the `master` branch

Unlock the branch by changing the number of required approving reviewers in the `master` branch rule back to 1.

### 4. Create and push a release tag

```sh
make tag RELEASE_VERSION=v1.3.0
git push upstream v1.3.0
```

### 5. Fast-forward the `latest` and release branches

The `latest` branch points to the latest release tag to keep the main website subdomain up-to-date.
Run the following commands to do so:

```sh
git checkout latest
git reset --hard tags/v1.3.0
git push -f upstream latest
```

Similarly, to update the release branch, run:

```sh
git checkout v1.3.x
git reset --hard tags/v1.3.0
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
git checkout v1.3.x
git pull
git checkout -b release-v1.3.1
```

Run the pre-release `make` target:

```sh
make prerelease RELEASE_VERSION=v1.3.1
```

The following changes should be present:

- `changelog/generated/v1.3.0.md`: commit changes (created by changelog generation).
- `changelog/fragments/*`: commit deleted fragment files (deleted by changelog generation).

Commit these changes and push:

```sh
git add --all
git commit -m "Release v1.3.1"
git push -u origin release-v1.3.1
```

### 2. Create and merge a new PR

Create and merge a new PR for the commit created in step 1. You can force-merge your PR to the locked-down `v1.3.x`
if you have admin access to the operator-sdk repo, or ask an administrator to do so.

### 3. Unlock the `v1.3.x` branch

Unlock the branch by changing the number of required approving reviewers in the `v.*` branch rule back to 1.

### 4. Create and push a release tag

```sh
make tag RELEASE_VERSION=v1.3.1
git push upstream v1.3.1
```

### 5. Fast-forward the `latest` branch

The `latest` branch points to the latest release tag to keep the main website subdomain up-to-date.
Run the following commands to do so:

```sh
git checkout latest
git reset --hard tags/v1.3.1
git push -f upstream latest
```

### 6. Post release steps

- Make an [operator-framework Google Group][of-ggroup] post.
- Post to Kubernetes slack in #kubernetes-operators and #operator-sdk-dev.
- In the [GitHub milestone][gh-milestones], bump any open issues to the following release.


## Further reading

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


[git]:https://git-scm.com/downloads
[gpg]:https://gnupg.org/download/
[gpg-key-create]:https://docs.github.com/en/free-pro-team@latest/github/authenticating-to-github/managing-commit-signature-verification
[gpg-upload]:https://www.gnupg.org/gph/en/manual/x457.html
[netlify-deploy]:https://docs.netlify.com/site-deploys/overview/#deploy-summary
[doc-owners]: https://github.com/operator-framework/operator-sdk/blob/master/OWNERS
[release-page]:https://github.com/operator-framework/operator-sdk/releases
[backports]:/docs/upgrading-sdk-version/backport-policy
[of-ggroup]:https://groups.google.com/g/operator-framework
[gh-milestones]:https://github.com/operator-framework/operator-sdk/milestones
[Makefile]:https://github.com/operator-framework/operator-sdk/blob/master/Makefile
