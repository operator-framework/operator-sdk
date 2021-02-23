---
title: Development
linkTitle: Development
weight: 1
---

## Installation

### Prerequisites

- [git][git-tool]
- [go][go-tool] version 1.15

### Download Operator SDK

Go to the [operator-sdk repo][repo-sdk] and follow the [fork guide][fork-guide] to fork and set up a local repository.

### Build the Operator SDK CLI

Build the Operator SDK CLI `operator-sdk` binary:

```sh
make install
```

## Testing

See the [testing][dev-testing] and [documentation][dev-docs] guides for more information.

## Releasing

See the [release guide][dev-release] for more information.

## Continuous Integration (CI)

The operator-sdk repo uses [Travis CI][travis] to test each pull request and build images for both master commits
and releases. You can alter these processes by modifying this [`.travis.yml`][travis.yml] config file.

### Testing builds with new architectures

Follow these steps to execute the Travis `Deploy` stage against your branch
to demonstrate that the merge build will complete as expected.

- Enable Travis in your fork repository. See [this guide][travis-setup] for more information.
- Create public image repos for each image built by `make image-build`; make sure the registry used supports
multi-arch images, like quay.io.
  - For each image type, you need one repo for the manifest list and one for each architecture being tested.
- Set each image variable (that ends in `_IMAGE`, not `_BASE_IMAGE`) found in the Makefile
as an environment variable in `.travis.yml`, ex. `export SCORECARD_TEST_IMAGE=<registry>/<username>/scorecard-test:latest`
- Create a PR with your configuration changes to _your_ fork, with the first commit message containing
`[travis deploy]`.
  - This commit is only for testing on your fork's PR. This commit/message should not be present in an operator-sdk
  repo PR.
- Ensure the image builds for that PR pass.
- Create a PR to the operator-sdk repo with a description containing a link to the Travis build page
showing a successful deployment stage with your changes.

[git-tool]:https://git-scm.com/downloads
[go-tool]:https://golang.org/dl/
[repo-sdk]:https://github.com/operator-framework/operator-sdk
[fork-guide]:https://help.github.com/en/articles/fork-a-repo
[dev-testing]: /docs/contribution-guidelines/testing
[dev-docs]: /docs/contribution-guidelines/documentation
[dev-release]: /docs/contribution-guidelines/releasing
[travis]: https://docs.travis-ci.com/
[travis.yml]: https://github.com/operator-framework/operator-sdk/blob/master/.travis.yml
[travis-setup]: https://docs.travis-ci.com/user/tutorial/#to-get-started-with-travis-ci-using-github
