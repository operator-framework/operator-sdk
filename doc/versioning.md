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

## Patch versions

Patch versions changes are meant only for bug fixes, and will not break compatibility of the current minor version. A patch release will contain a collection of minor bug fixes, or individual major and security bug fixes, depending on severity.

[link-semver]:https://semver.org/
[link-github-milestones]: https://help.github.com/en/articles/about-milestones
