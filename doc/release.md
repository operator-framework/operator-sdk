# Releases

Making an Operator SDK release involves:

- Updating `CHANGELOG.md`.
- Tagging and signing a git commit and pushing the tag to GitHub.
- Building a release binary and signing the binary
- Creating a release by uploading binary, signature, and `CHANGELOG.md` updates for the release to GitHub.
- Creating a patch version branch of the form `v1.2.x` for major and minor releases.

Releases can only be performed by [maintainers][doc-maintainers].

## Dependency and platform support

### Kubernetes versions

As the Operator SDK interacts directly with the Kubernetes API, certain API features are assumed to exist in the target cluster. The currently supported Kubernetes version will always be listed in the SDK [prerequisites section][doc-kube-version].

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

Release notes should be a thorough description of changes made to code, documentation, and design. Individual changes, such as bug fixes, should be given their own bullet point with a short description of what was changed. Issue links and handle of the developer who worked on the change should be included whenever possible.

The following is the format for major and minor releases:

```Markdown
[Short description of release (ex. reason, theme)]

### Features
- [Short description of feature] (#issue1, #issue2, ..., @maintainer_handle)
...

### Bug fixes
- [Short description of fix] (#issue1, #issue2, ..., @maintainer_handle)
...

### Miscellaneous
- [Short description of change] (#issue1, #issue2, ..., @maintainer_handle)
...
```

Patch releases should have the following format:

```Markdown
[Medium-length description of release (if not complex, short is fine); explanation required]

### Bug fixes
- [Short description of fix] (#issue1, #issue2, ..., @maintainer_handle)
...
```

## Release Signing

When a new release is created, the tag for the commit it signed with a maintainers' gpg key and
the binaries for the release are also signed by the same key. All keys used by maintainers will
be available via public PGP keyservers such as [pgp.mit.edu][mit-keyserver].

For new maintainers who have not done a release and do not have their PGP key on a public
keyserver, output your armored public key using this command:

```sh
$ gpg --armor --export "$GPG_EMAIL" > mykey.asc
```

Then, copy and paste the content of the outputted file into the `Submit a key` section on
the [MIT PGP Public Key Server][mit-keyserver] or any other public keyserver that forwards
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

# Release steps

These steps describe how to conduct a release of the SDK, upgrading from `v1.2.0` to `v1.3.0`. Replace these versions with the current and new version you are releasing, respectively.

**Note:** `master` should be frozen between steps 1 and 3 so that all commits will be either in the new release or have a pre-release version, ex. `v1.2.0+git`. Otherwise commits might be built into a release that shouldn't or have an incorrect version, which makes debugging user issues difficult.

## (Path release only) Cherry-picking to a release branch

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

Commit changes to the following four files:
* `version/version.go`: update `Version` to `v1.3.0`.
* `pkg/scaffold/gopkgtoml.go`: under the `[[constraint]]` for `github.com/operator-framework/operator-sdk`, comment out `branch = "master"`, uncomment `version = "v1.2.0"`, and change `v1.2.0` to `v1.3.0`.
* `pkg/scaffold/gopkgtoml_test.go`: same as for `pkg/scaffold/gopkgtoml.go`.
* `CHANGELOG.md`: update the `## Unreleased` header to `## v1.3.0`.

Create a new PR for `release-v1.3.0`.

**Note:** CI will not pass for this commit because the new `version` does not exist yet in Git history.

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
* `version/version.go`: update `Version` to `v1.3.0+git`.
* `pkg/scaffold/gopkgtoml.go`: under the `[[constraint]]` for `github.com/operator-framework/operator-sdk`, comment out `version = "v1.3.0"` and uncomment `branch = "master"`.
* `pkg/scaffold/gopkgtoml_test.go`: same as for `pkg/scaffold/gopkgtoml.go`.
* `CHANGELOG.md`: add the following as a new set of headers above `## v1.3.0`:
    ```
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

You've now fully released a new version of the Operator SDK. Good work!

[doc-maintainers]:../MAINTAINERS
[doc-git-default-key]:https://help.github.com/articles/telling-git-about-your-signing-key/
[doc-gpg-default-key]:https://lists.gnupg.org/pipermail/gnupg-users/2001-September/010163.html
[link-github-gpg-key-upload]:https://github.com/settings/keys
[link-git-config-gpg-key]:https://git-scm.com/book/en/v2/Git-Tools-Signing-Your-Work
[doc-kube-version]:https://github.com/operator-framework/operator-sdk#prerequisites
[mit-keyserver]:https://pgp.mit.edu
[release-page]:https://github.com/operator-framework/operator-sdk/releases
