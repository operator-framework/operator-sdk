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
$ MAV_MIN_VER="v${MAJOR_VERSION}.${NEW_MINOR_VERSION}"
$ git checkout -b "${MAJ_MIN_VER}.x" tags/"v${MAJ_MIN_VER}.0"
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
- Building a release binary and uploading the binary to GitHub.

Releases can only be performed by [maintainers][doc-maintainers].

## Dependency and platform support

### Kubernetes versions

As the Operator SDK interacts directly with the Kubernetes API, certain API features are assumed to exist in the target cluster. The currently supported Kubernetes version will always be listed in the SDK [prerequisites section][doc-kube-version].

### Operating systems and architectures

Release binaries will be built for the `x86_64` architecture for both GNU Linux and MacOS Darwin platforms.

Support for the Windows platform or any architecture other than `x86_64` is not on the roadmap at this time.

## Binaries

Creating release binaries:
```bash
$ ./release.sh "v${VERSION}"
```

## Release tags

Every release will have a corresponding git tag.

Make sure you've [uploaded your GPG key][link-github-gpg-key-upload] and configured git to [use that signing key][link-git-config-gpg-key] either globally or for the Operator SDK repository. Note: the email the key is issued for must be the email you use for git.

```bash
$ git config [--global] user.signingkey "$GPG_KEY_ID"
$ git config [--global] user.email "$GPG_EMAIL"
```

Tagging will be handled by `release.sh`.

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

[link-semver]:https://semver.org/
[link-github-milestones]: https://help.github.com/articles/about-milestones/
[doc-maintainers]:../MAINTAINERS
[link-github-gpg-key-upload]:https://github.com/settings/keys
[link-git-config-gpg-key]:https://git-scm.com/book/en/v2/Git-Tools-Signing-Your-Work
[doc-kube-version]:https://github.com/operator-framework/operator-sdk#prerequisites