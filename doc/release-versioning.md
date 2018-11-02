# Versioning

The following is a concise explanation of how Operator SDK versions are determined. The Operator SDK versioning follows [semantic versioning][link-semver] standards.

## Milestones

Operator SDK [milestones][link-github-milestones] represent changes to the SDK spanning multiple issues, such as a design change. Milestones help SDK developers determine how close we are to new releases, either major or minor; a milestone must be completed before a new version is released. Milestones and their involved issues are determined by maintainers.

Milestone labels have the form: `milestone-x.y.0`, where `x` and `y` are major and minor SDK versions, respectively. This particular milestone demarcates the SDK `x.y.0` release; once issues within this milestone are addressed, the release process can begin.

## Major versions

Major version changes can break compatibility between the previous major versions; they are not necessarily backwards or forwards compatible. SDK change targets include but are not limited to:

- `operator-sdk` command and sub-commands
- Golang API
- Formats of various yaml manifest files

## Minor versions

Minor version changes will not break compatibility between the previous minor versions; to do so is a bug. SDK changes will involve addition of optional features, non-breaking enhancements, and *minor* bug fixes identified from previous versions.

### Creating a minor version branch

We expect to release patches for minor releases, so we create a patch trunk to branch from. The naming convention follows "v2.1.x", where the major version is 2, minor is 1, and "x" is a patch version placeholder.

```bash
$ export MAJ_MIN_VER="${MAJOR_VERSION}.${NEW_MINOR_VERSION}"
$ git checkout -b "v${MAJ_MIN_VER}.x" tags/"v${MAJ_MIN_VER}.0"
$ git push git@github.com:operator-framework/operator-sdk.git "v${MAJ_MIN_VER}.x"
```

## Patch versions

Patch versions changes are meant only for bug fixes, and will not break compatibility of the current minor version. A patch release will contain a collection of minor bug fixes, or individual major and security bug fixes, depending on severity.

### Creating a patch version branch

As more than one patch may be created per minor release, patch branch names of the form "${MAJOR_VERSION}.${MINOR_VERSION}.${PATCH_VERSION}" will be created after a bug fix has been pushed, and the bug fix branch merged into the patch branch only after testing.

```bash
$ git checkout "v${MAJOR_VERSION}.${MINOR_VERSION}.x"
$ git checkout -b "cherry-picked-change"
$ git cherry-pick "$GIT_COMMIT_HASH"
$ git push origin "cherry-picked-change"
```

# Releases

Making an Operator SDK release involves:

- Tagging and signing a git commit and pushing the tag to GitHub.
- Building a release binary, signing the binary, and uploading both binary and signature to GitHub.

Releases can only be performed by [maintainers][doc-maintainers].

## Dependency and platform support

### Kubernetes versions

As the Operator SDK interacts directly with the Kubernetes API, certain API features are assumed to exist in the target cluster. The currently supported Kubernetes version will always be listed in the SDK [prerequisites section][doc-kube-version].

### Operating systems and architectures

Release binaries will be built for the `x86_64` architecture for both GNU Linux and MacOS Darwin platforms.

Support for the Windows platform or any architecture other than `x86_64` is not on the roadmap at this time.

## Binaries and signatures

Binaries will be signed using a maintainers' verified GitHub PGP key. Both binary and signature will be uploaded to the release. Ensure you import maintainer keys to verify release binaries.

Creating release binaries and signatures:

```bash
$ ./release.sh "v${VERSION}"
```

**Note**: you must have both [`git`][doc-git-default-key] and [`gpg`][doc-gpg-default-key] default PGP keys set locally for `release.sh` to run without error.

## Release tags

Every release will have a corresponding git tag.

Make sure you've [uploaded your GPG key][link-github-gpg-key-upload] and configured git to [use that signing key][link-git-config-gpg-key] either globally or for the Operator SDK repository. Tagging will be handled by `release.sh`.

**Note**: the email the key is issued for must be the email you use for git.

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

# Release Signing

When a new release is created, the tag for the commit it signed with a maintainer's gpg key and
the binaries for the release are also signed by the same key. All keys used by maintainers will
be available via public gpg keyservers such as [pgp.mit.edu][mit-keyserver]. To verify a git
tag signature, use this command:

```sh
$ git verify-tag --verbose ${TAG_NAME}
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
gpg: Can't check signature: public key not found
```

To download the key, use this command, replacing `${KEY_ID}` with the RSA key string provided in the output
of the previous command:

```sh
$ gpg --recv-key ${KEY_ID}
```

Now you should be able to verify the tags and/or binaries

## For maintainers

For new maintainers who have not done a release and do not have their gpg key on a public
keyserver, you must add your public key to a keyserver. To do this, output your armored
public key using this command:

```sh
$ gpg --armor --export ${GPG_EMAIL} > mykey.asc
```

Then, copy and paste the content of the outputted file into the `Submit a key` section on
the [MIT PGP Public Key Server][mit-keyserver] or any other public keyserver that forwards
the key to other public keyservers. Once that is done, other people can download your public
key and you are ready to sign releases.

[link-semver]:https://semver.org/
[link-github-milestones]: https://help.github.com/articles/about-milestones/
[doc-maintainers]:../MAINTAINERS
[doc-git-default-key]:https://help.github.com/articles/telling-git-about-your-signing-key/
[doc-gpg-default-key]:https://lists.gnupg.org/pipermail/gnupg-users/2001-September/010163.html
[link-github-gpg-key-upload]:https://github.com/settings/keys
[link-git-config-gpg-key]:https://git-scm.com/book/en/v2/Git-Tools-Signing-Your-Work
[doc-kube-version]:https://github.com/operator-framework/operator-sdk#prerequisites
[mit-keyserver]:https://pgp.mit.edu
