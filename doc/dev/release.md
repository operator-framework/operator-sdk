# Releases

Making an Operator SDK release involves:

- Updating `CHANGELOG.md`.
- Tagging and signing a git commit and pushing the tag to GitHub.
- Building a release binary and signing the binary
- Creating a release by uploading binary, signature, and `CHANGELOG.md` updates for the release to GitHub.
- Creating a patch version branch of the form `v1.2.x` for major and minor releases.

Releases can only be performed by [maintainers][doc-maintainers].

## Dependency and platform support

### Go version

Release binaries will be built with the Go compiler version specified in the Operator SDK's [prerequisites section][doc-readme-prereqs].

### Kubernetes versions

As the Operator SDK interacts directly with the Kubernetes API, certain API features are assumed to exist in the target cluster. The currently supported Kubernetes version will always be listed in the SDK [prerequisites section][doc-readme-prereqs].

### Operating systems and architectures

Release binaries will be built for the `x86_64` architecture for both GNU Linux and MacOS Darwin platforms.

Support for the Windows platform or any architecture other than `x86_64` is not on the roadmap at this time.

## Binaries and signatures

Binaries will be signed using a maintainers' verified GitHub PGP key. Both binary and signature will be uploaded to the release. Ensure you import maintainer keys to verify release binaries.

## Release tags

Every release will have a corresponding git semantic version tag beginning with `v`, ex. `v1.2.3`.

Make sure you've [uploaded your GPG key][link-github-gpg-key-upload] and configured git to [use that signing key][link-git-config-gpg-key] either globally or for the Operator SDK repository. Tagging will be handled by `release.sh`.

**Note:** the email the key is issued for must be the email you use for git.

```bash
$ git config [--global] user.signingkey "$GPG_KEY_ID"
$ git config [--global] user.email "$GPG_EMAIL"
```

## Release Notes

Release notes should thoroughly describe changes made to code, documentation, and design of the SDK. PR links should be included wherever possible.

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

To verify a git tag signature, use this command:

```sh
$ git verify-tag --verbose "$TAG_NAME"
```

To verify a release binary using the provided asc files, place the binary and corresponding asc
file into the same directory and use the corresponding command:

```sh
# macOS
$ gpg --verify operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin.asc
# GNU/Linux
$ gpg --verify operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu.asc
```

If you do not have the maintainers public key on your machine, you will get an error message similar
to this:

```sh
$ git verify-tag ${TAG_NAME}
gpg: Signature made Wed 31 Oct 2018 02:57:31 PM PDT
gpg:                using RSA key 4AEE18F83AFDEB23
gpg: Cant check signature: public key not found
```

To download the key, use this command, replacing `$KEY_ID` with the RSA key string provided in the output
of the previous command:

```sh
$ gpg --recv-key "$KEY_ID"
```

Now you should be able to verify the tags and/or binaries.

## Release steps

These steps describe how to conduct a release of the SDK, upgrading from `v1.2.0` to `v1.3.0`. Replace these versions with the current and new version you are releasing, respectively.

**Note:** `master` should be frozen between steps 1 and 3 so that all commits will be either in the new release or have a pre-release version, ex. `v1.2.0+git`. Otherwise commits might be built into a release that shouldn't or have an incorrect version, which makes debugging user issues difficult.

### (Patch release only) Cherry-picking to a release branch

As more than one patch may be created per minor release, branch names of the form `v1.3.x` are created after a minor version is released. Bug fixes will be merged into the release branch only after testing.

Add fixes to the release branch by doing the following:

```bash
$ git checkout v1.3.x
$ git checkout -b release-v1.3.1
$ git cherry-pick "$GIT_COMMIT_HASH" # Usually from master
$ git push origin release-v1.3.1
```

Create a PR from `release-v1.3.1` to `v1.3.x`. Once CI passes and your PR is merged, continue to step 1.

### 1. Create a PR for release version and CHANGELOG.md updates

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

Commit changes to the following six files:

- `version/version.go`: update `Version` to `v1.3.0`.
- `internal/pkg/scaffold/gopkgtoml.go`, under the `[[constraint]]` for `github.com/operator-framework/operator-sdk`:
  - Comment out `branch = "master"`
  - Un-comment `version = "v1.2.0"`
  - Change `v1.2.0` to `v1.3.0`
- `internal/pkg/scaffold/gopkgtoml_test.go`: same as for `internal/pkg/scaffold/gopkgtoml.go`.
- `internal/pkg/scaffold/ansible/gopkgtoml.go`: same as for `internal/pkg/scaffold/gopkgtoml.go`.
- `internal/pkg/scaffold/helm/gopkgtoml.go`: same as for `internal/pkg/scaffold/gopkgtoml.go`.
- `CHANGELOG.md`: update the `## Unreleased` header to `## v1.3.0`.

Create a new PR for `release-v1.3.0`.

### 2. Create a release tag, binaries, and signatures

The top-level `release.sh` script will take care of verifying versions in files described in step 1, and tagging and verifying the tag, as well as building binaries and generating signatures by calling `make release`.

Call the script with the only argument being the new SDK version:

```sh
$ ./release.sh v1.3.0
```

Release binaries and signatures will be in `build/`. Both binary and signature file names contain version, architecture, and platform information; signature file names correspond to the binary they were generated from suffixed with `.asc`. For example, signature file `operator-sdk-v1.3.0-x86_64-apple-darwin.asc` was generated from a binary named `operator-sdk-v1.3.0-x86_64-apple-darwin`. To verify binaries and tags, see the [verification section](#verifying-a-release).

**Note:** you must have both [`git`][doc-git-default-key] and [`gpg`][doc-gpg-default-key] default PGP keys set locally for `release.sh` to run without error. Additionally you must add your PGP key to a [public-key-server](#release-signing).

Push tag `v1.3.0` upstream:

```sh
$ git push --tags
```

Once this tag passes CI, go to step 3. For more info on tagging, see the [release tags section](#release-tags).

**Note:** If CI fails for some reason, you will have to revert the tagged commit, re-commit, and make a new PR.

### 3. Create a PR for post-release version and CHANGELOG.md updates

Check out a new branch from master (or use your `release-v1.3.0`) and commit the following changes:

- `version/version.go`: update `Version` to `v1.3.0+git`.
- `internal/pkg/scaffold/gopkgtoml.go`, under the `[[constraint]]` for `github.com/operator-framework/operator-sdk`:
  - Comment out `version = "v1.3.0"`
  - Un-comment `branch = "master"`
- `internal/pkg/scaffold/gopkgtoml_test.go`: same as for `internal/pkg/scaffold/gopkgtoml.go`.
- `internal/pkg/scaffold/ansible/gopkgtoml.go`: same as for `internal/pkg/scaffold/gopkgtoml.go`.
- `internal/pkg/scaffold/helm/gopkgtoml.go`: same as for `internal/pkg/scaffold/gopkgtoml.go`.
- `CHANGELOG.md`: add the following as a new set of headers above `## v1.3.0`:

    ```markdown
    ## Unreleased

    ### Added

    ### Changed

    ### Deprecated

    ### Removed

    ### Bug Fixes
    ```

Create a new PR for this branch. Once this PR passes CI and is merged, `master` can be unfrozen.

### 4. Releasing binaries, signatures, and release notes

The final step is to upload binaries, their signature files, and release notes from `CHANGELOG.md`.

**Note:** if this is a pre-release, make sure to check the `This is a pre-release` box under the file attachment frame. If you are not sure what this means, ask another maintainer.

1. Go to the SDK [release page][release-page] and click the `Draft a new release` button in the top right corner.
1. Select the tag version `v1.3.0`, and set the title to `v1.3.0`.
1. Copy and paste any `CHANGELOG.md` under the `v1.3.0` header that have any notes into the description form.
1. Attach all binaries and `.asc` signature files to the release by dragging and dropping them.
1. Click the `Publish release` button.

You've now fully released a new version of the Operator SDK. Good work! However, there is one more step that needs to be completed: making a release branch to allow us to make patch fixes for this release.

### 5. Making a new release branch

If you have created a new major or minor release, you need to make a new branch for it. To do this, checkout the tag that you created and make a new branch that matches the version you released with `x` in the position of the patch number. For example, to make a new release branch after `v1.3.0` and push it to the repo, you would follow these steps:

```console
$ git checkout tags/v1.3.0
Note: checking out 'tags/v1.3.0'.
...
$ git checkout -b v1.3.x
Switched to a new branch 'v1.3.x'
$ git push origin v1.3.x
Total 0 (delta 0), reused 0 (delta 0)
remote:
remote: Create a pull request for 'v1.3.x' on GitHub by visiting:
remote:      https://github.com/operator-framework/operator-sdk/pull/new/v1.3.x
remote:
To github.com:operator-framework/operator-sdk.git
 * [new branch]      v1.3.x -> v1.3.x
```

Now that the branch exists, you need to make the post-release PR for the new release branch. To do this, simply follow the same steps as in [step 3](#3-create-a-pr-for-post-release-version-and-changelogmd-updates) with the addition of changing the branch name in the `gopkgtoml` scaffold from `master` to the new branch (for example, `v1.3.x`). Then, make the PR against the new branch.

[doc-maintainers]:../../MAINTAINERS
[doc-readme-prereqs]:../../README.md#prerequisites
[doc-git-default-key]:https://help.github.com/en/articles/telling-git-about-your-signing-key
[doc-gpg-default-key]:https://lists.gnupg.org/pipermail/gnupg-users/2001-September/010163.html
[link-github-gpg-key-upload]:https://github.com/settings/keys
[link-git-config-gpg-key]:https://git-scm.com/book/en/v2/Git-Tools-Signing-Your-Work
[doc-changelog]:../../CHANGELOG.md
[release-page]:https://github.com/operator-framework/operator-sdk/releases
