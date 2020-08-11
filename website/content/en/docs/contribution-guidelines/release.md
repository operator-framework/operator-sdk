---
title: Release Guide
weight: 30
---

Making an Operator SDK release involves:

- Updating `CHANGELOG.md` and migration guide.
- Tagging and signing a git commit and pushing the tag to GitHub.
- Building a release binary and signing the binary
- Creating a release by uploading binary, signature, and `CHANGELOG.md` updates for the release to GitHub.
- Creating a release branch of the form `v1.2.x` for each major and minor release.

Release steps can be found [below](#release-steps). If you have not run a release before we recommend reading
through the sections directly following this one.

## Dependency and platform support

### Go version

Release binaries will be built with the Go compiler version specified in the Operator SDK's [prerequisites section][doc-readme-prereqs].

### Kubernetes versions

As the Operator SDK interacts directly with the Kubernetes API, certain API features are assumed to exist in the target cluster. The currently supported Kubernetes version will always be listed in the SDK [prerequisites section][doc-readme-prereqs].

### Operating systems and architectures

Release binaries will be built for the `x86_64` architecture for MacOS Darwin platform and for the following GNU Linux architectures: `x86_64`, `ppc64le`, `arm64`, and `s390x`.

Base images for ansible-operator, helm-operator, and scorecard-test will be built for the following GNU Linux architectures: `x86_64`, `ppc64le`, and `arm64`.

Support for the Windows platform is not on the roadmap at this time.

## Binaries and signatures

Binaries will be signed using a maintainers' verified GitHub PGP key. Both binary and signature will be uploaded to the release. Ensure you import maintainer keys to verify release binaries.

## Release tags

Every release will have a corresponding git semantic version tag beginning with `v`, ex. `v1.2.3`.

Make sure you've [uploaded your GPG key][link-github-gpg-key-upload] and configured git to [use that signing key][link-git-config-gpg-key] either globally or for the Operator SDK repository. Tagging will be handled by `release.sh`.

**Note:** the email the key is issued for must be the email you use for git.

```sh
$ git config [--global] user.signingkey "$GPG_KEY_ID"
$ git config [--global] user.email "$GPG_EMAIL"
```

Also, make sure that you setup the git gpg config as follows.
```console
$ cat ~/.gnupg/gpg.conf
default-key $GPG_KEY_ID
```

**NOTE** If you do a release from an OSX machine, you need to configure `gnu-gpg` to sign the release's tag:
- Install the requirements by running: `brew install gpg2 gnupg pinentry-mac`
- Append the following to your ~/.bash_profile or ~/.bashrc or ~/.zshrc
```sh
export GPG_TTY=`tty`
```
- Restart your Terminal or source your ~/.\*rc file
- Then, make sure git uses gpg2 and not gpg
```sh
$ git config --global gpg.program gpg2
```
- To make sure gpg2 itself is working
```sh
$ echo "test" | gpg2 --clearsign
```

## Release branches

Each minor release has a corresponding release branch of the form `vX.Y.x`, where `X` and `Y` are the major and minor
release version numbers and the `x` is literal. This branch accepts bug fixes according to our [backport policy][backports].

This branch must be created before the release occurs to appease the Netlify website configuration demons.
You can do so by running the following before proceeding with the release, assuming the upstream SDK is the `origin` remote repo:

```sh
$ git checkout master
$ git pull
$ git checkout -b v1.3.x
$ git push -u origin v1.3.x
```

After the minor release is made, this branch must be fast-forwarded to that release's tag and a post-release PR made
against this branch. See the [release process](#4-create-a-pr-for-post-release-version-updates) for more details.

#### Cherry-picking

Once a minor release is complete, bug fixes can be merged into the release branch for the next patch release.
Fixes can be added automatically by posting a `/cherry-pick v1.3.x` comment in the `master` PR, or manually by running:

```sh
$ git checkout v1.3.x
$ git checkout -b cherrypick/some-bug
$ git cherry-pick "$GIT_COMMIT_HASH" # Hash of the merge commit to master.
$ git push origin cherrypick/some-bug
```

Create and merge a PR from your branch to `v1.3.x`.

## GitHub release information

### Locking down branches

Once a release PR has been made and all tests pass, the SDK's `master` branch, or [release branch](#release-branches) for patch releases,
should be locked so commits cannot happen between the release PR and release tag push. To lock down a branch:

1. Go to `Settings -> Branches` in the SDK repo.
1. Under `Branch protection rules`, click `Edit` on the `master` or release branches rule.
1. In section `Protect matching branches` of the `Rule settings` box, increase the number of required approving reviewers to its maximum allowed value.

Now only administrators (maintainers) should be able to force merge PRs. Make sure everyone in the relevant Slack channel is aware of the release so they do not force merge by accident.

Unlock `master` or release branch after the release has completed (after step 3 is complete) by changing the number of required approving reviewers back to 1.

### Releasing

The GitHub [`Releases` tab][release-page] in the operator-sdk repo is where all SDK releases live. To create a GitHub release:

1. Go to the SDK [`Releases` tab][release-page] and click the `Draft a new release` button in the top right corner.
1. Select the tag version `v1.3.0`, and set the title to `v1.3.0`.
1. Copy and paste `CHANGELOG.md` updates under the `v1.3.0` header into the description form (see [below](#release-notes)).
1. Attach all binaries and `.asc` signature files to the release by dragging and dropping them.
1. Click the `Publish release` button.

**Note:** if this is a pre-release, make sure to check the `This is a pre-release` box under the file attachment frame. If you are not sure what this means, ask another maintainer.

#### Release notes

GitHub release notes should thoroughly describe changes made to code, documentation, and design of the SDK. PR links should be included wherever possible.

The following sections, often directly copied from our [changelog][doc-changelog], are used as release notes:

```Markdown
[Version as title, ex. v1.2.3]

### Added
- [Short description of feature added] (#PR)
...

### Changed
- [Short description of change made] (#PR)
...

### Deprecated
- [Short description of feature deprecated] (#PR)
...

### Removed
- [Short description of feature removed] (#PR)
...

### Bug Fixes
- [Short description of bug and fix] (#PR)
...
```

## Release Signing

When a new release is created, the tag for the commit it signed with a maintainers' gpg key and
the binaries for the release are also signed by the same key. All keys used by maintainers will
be available via public PGP keyservers such as pool.sks-keyservers.net.

For new maintainers who have not done a release and do not have their PGP key on a public
keyserver, output your armored public key using this command:

```sh
$ gpg --armor --export "$GPG_EMAIL" > mykey.asc
```

Then, copy and paste the content of the outputted file into the `Submit a key` section on
pool.sks-keyservers.net or any other public keyserver that synchronizes
the key to other public keyservers. Once that is done, other people can download your public
key and you are ready to sign releases.

## Verifying a release

To verify a git tag, use this command:

```sh
$ git verify-tag --verbose "$TAG_NAME"
```

If you do not have the mantainers public key on your machine, you will get an error message similiar to this:

```console
$ git verify-tag --verbose "$TAG_NAME"
object 61e0c23e9d2e217f8d95ac104a8f2545c102b5c3
type commit
tag v0.6.0
tagger Ish Shah <ishah@redhat.com> 1552688145 -0700

Operator SDK v0.6.0
gpg: Signature made Fri Mar 15 23:15:45 2019 CET
gpg:                using RSA key <KEY_ID>
gpg: Can't check signature: No public key
```

To download the key, use the following command, replacing `$KEY_ID` with the RSA key string provided in the output of the previous command:

```sh
$ gpg --recv-key "$KEY_ID"
```

To verify a release binary using the provided asc files see the [installation guide.][install-guide]

## Release steps

These steps describe how to conduct a release of the SDK, upgrading from `v1.2.0` to `v1.3.0`.
Replace these versions with the current and new version you are releasing, respectively.

For major and minor releases, `master` should be locked between steps 1 and 3 so that all commits will be either in the new release
or have a pre-release version, ex. `v1.2.0+git`. Otherwise commits might be built into a release that shouldn't be.
For patch releases, ensure all required bugs are [cherry-picked](#cherry-picking), then the release branch `v1.3.x` should be locked down.

**Important:** ensure a release branch-to-subdomain mapping exists in the SDK's Netlify configuration _prior to creating a release_,
ex. `v1.3.x` to `https://v1-3-x.sdk.operatorframework.io`. You can ping SDK [approvers][doc-owners] to ensure a
[release branch](#release-branches) is created prior to the release and that this mapping is created.

### 1. Create a PR for release version, CHANGELOG.md, and migration guide updates

Once all PR's needed for a release have been merged, branch from `master`:

```sh
$ git checkout master
$ git pull
```

If making a patch release, check out the corresponding minor version branch:

```sh
$ git checkout v1.2.x
$ git pull
```

Create a new branch to push release commits:

```sh
$ git checkout -b release-v1.3.0
```

Run the CHANGELOG and migration guide generator:

```sh
$ GEN_CHANGELOG_TAG=v1.3.0 make gen-changelog
```

Commit the following changes:

- `internal/version/version.go`: update `Version` to `v1.3.0`.
- `website/content/en/docs/installation/install-operator-sdk.md`: update the linux and macOS URLs to point to the new release URLs.
- `CHANGELOG.md`: commit changes (updated by changelog generation).
- `website/content/en/docs/upgrading-sdk-version/v1.3.0.md`: commit changes (created by changelog generation).
- `changelog/fragments/*`: commit deleted fragment files (deleted by changelog generation).
- **(Major and minor releases only)** `website/config.toml`: update `version_menu = "Releases"` with the patch-less version string `version_menu = "v1.3"`,
and add the following lines under `[[params.versions]]` for `master`:
  ```toml
  [[params.versions]]
    version = "v1.3"
    url = "https://v1-3-x.sdk.operatorframework.io"
  ```

---

Create and merge a new PR for `release-v1.3.0`. Once this PR is merged, lock down the master or release branch
to prevent further commits between this and step 4. See [this section](#locking-down-branches) for steps to do so.

### 2. Create a release tag, binaries, and signatures

The top-level `release.sh` script will take care of verifying versions in files described in step 1, and tagging and verifying the tag, as well as building binaries and generating signatures by calling `make release`.

Prerequisites:
- [`git`][doc-git-default-key] and [`gpg`][doc-gpg-default-key] default PGP keys are set locally.
- Your PGP key is publicly available in a [public key server](#release-signing).
- _For macOS users:_ GNU `sed` and `make` which are not installed by default. Install them with
  ```sh
  $ brew install gnu-sed make
  ```
  then ensure they are present in your `$PATH`.

Call the script with the only argument being the new SDK version:

```sh
$ ./release.sh v1.3.0
```

`operator-sdk` release binaries and signatures will be in `build/`. Both binary and signature file names contain version, architecture,
and platform information; signature file names correspond to the binary they were generated from suffixed with `.asc`.
For example, signature file `operator-sdk-v1.3.0-x86_64-apple-darwin.asc` was generated from a binary named `operator-sdk-v1.3.0-x86_64-apple-darwin`.
To verify binaries and tags, see the [verification section](#verifying-a-release).

<!-- TODO: remove when ansible/helm operator repos are created and code removed from this repo -->
`ansible-operator` and `helm-operator` release binaries and signatures are similarly built for upload so `make run`
can download them in their respective operator type projects. See [#3327](https://github.com/operator-framework/operator-sdk/issues/3327) for details.

Push tag `v1.3.0` upstream, assuming `origin` is the name of the upstream remote:

```sh
$ git push origin v1.3.0
```

Once this tag passes CI, go to step 3. For more info on tagging, see the [release tags section](#release-tags).

**Note:** If CI fails for some reason, you will have to revert the tagged commit, re-commit, and make a new PR.

### 3. Fast-forward the `latest` and release branches

The `latest` branch points to the latest release tag to keep the main website subdomain up-to-date.
Run the following commands to do so:

```sh
$ git checkout latest
$ git reset --hard tags/v1.3.0
$ git push -f origin latest
```

Similarly, to update the release branch, run:

```sh
$ git checkout v1.3.x
$ git reset --hard tags/v1.3.0
$ git push -f origin v1.3.x
```

### 4. Create a PR for post-release version updates

Check out a new branch from `master` or release branch and commit the following changes:

- `internal/version/version.go`: update `Version` to `v1.3.0+git`.
- **(Major and minor releases only)** `website/config.toml`: update `version_menu = "v1.3"` to `version_menu = "Releases"`.

---

Create a new PR for this branch targeting the `master` or release branch.

### 5. Releasing binaries, signatures, and release notes

The final step is to upload binaries, their signature files, and release notes from `CHANGELOG.md` for `v1.3.0`.
See [this section](#releasing) for steps to do so.

Unlock the `master` or release branch after the Github release is complete.
See [this section](#locking-down-branches) for steps to do so.

---

You've now fully released a new version of the Operator SDK. Good work! Make sure to follow the post-release steps below.

### (Post-release) Updating the operator-sdk-samples repo

Many releases change SDK API's and conventions, which are not reflected in the [operator-sdk-samples repo][sdk-samples-repo]. The samples repo should be updated and versioned after each SDK major/minor release with the same tag, ex. `v1.3.0`, so users can refer to the correct operator code for that release.

The release process for the samples repo is simple:

1. Make changes to all relevant operators (at least those referenced by SDK docs) based on API changes for the new SDK release.
1. Ensure the operators build and run as expected (see each operator's docs).
1. Once all API changes are in `master`, create a release tag locally:
    ```sh
    $ git checkout master && git pull
    $ export VER="v1.3.0"
    $ git tag --sign --message "Operator SDK Samples $VER" "$VER"
    ```
1. Push the tag to the remote, assuming `origin` is the name of the upstream remote:
    ```sh
    $ git push origin $VER
    ```

### (Post-release) Updating the release notes

Add the following line to the top of the GitHub release notes for `v1.3.0`:

```md
**NOTE:** ensure the `v1.3.0` tag is referenced when referring to sample code in the [SDK Operator samples repo](https://github.com/operator-framework/operator-sdk-samples/tree/v1.3.0) for this release. Links in SDK documentation are currently set to the samples repo `master` branch.
```

[install-guide]: /docs/installation/install-operator-sdk
[doc-maintainers]: https://github.com/operator-framework/operator-sdk/blob/master/MAINTAINERS
[doc-owners]: https://github.com/operator-framework/operator-sdk/blob/master/OWNERS
[doc-readme-prereqs]: /docs/installation/install-operator-sdk#prerequisites
[doc-git-default-key]:https://help.github.com/en/articles/telling-git-about-your-signing-key
[doc-gpg-default-key]:https://lists.gnupg.org/pipermail/gnupg-users/2001-September/010163.html
[link-github-gpg-key-upload]:https://github.com/settings/keys
[link-git-config-gpg-key]:https://git-scm.com/book/en/v2/Git-Tools-Signing-Your-Work
[doc-changelog]: https://github.com/operator-framework/operator-sdk/blob/master/CHANGELOG.md
[backports]:/docs/upgrading-sdk-version/backport-policy
[release-page]:https://github.com/operator-framework/operator-sdk/releases
[homebrew]:https://brew.sh/
[homebrew-formula]:https://github.com/Homebrew/homebrew-core/blob/master/Formula/operator-sdk.rb
[homebrew-readme]:https://github.com/Homebrew/homebrew-core/blob/master/CONTRIBUTING.md#to-submit-a-version-upgrade-for-the-foo-formula
[homebrew-repo]:https://github.com/Homebrew/homebrew-core
[sdk-samples-repo]:https://github.com/operator-framework/operator-sdk-samples
