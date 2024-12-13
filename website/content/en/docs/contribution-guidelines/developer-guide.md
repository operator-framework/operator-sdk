---
title: Development
linkTitle: Development
weight: 1
---

## Installation

### Prerequisites

- [git][git-tool]
- [go][go-tool] version 1.23

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

The operator-sdk repo uses [Github Actions][sdk-actions] to test each pull request and build images for both master commits
and release tags. You can alter these processes by modifying the appropriate [Action config][sdk-action-cfgs].

### Adding new architectures

The operator-sdk project builds binaries for [several os's/architectures][readme-platforms].
If you wish to add support for a new one, please create a feature request issue before
implementing support for that platform and submitting a PR.

If you'd like to implement support yourself, you can test a new architecture by enabling Actions
in your repository, add a platform pair to the [`deploy`][deploy-workflow] workflow's `build and push` step,
and push to your main branch. Once the updated Action passes, submit a PR linking the passing Action run.


[git-tool]:https://git-scm.com/downloads
[go-tool]:https://golang.org/dl/
[repo-sdk]:https://github.com/operator-framework/operator-sdk
[fork-guide]:https://docs.github.com/en/get-started/quickstart/fork-a-repo
[dev-testing]: /docs/contribution-guidelines/testing
[dev-docs]: /docs/contribution-guidelines/documentation
[dev-release]: /docs/contribution-guidelines/releasing
[sdk-actions]:https://github.com/operator-framework/operator-sdk/actions
[sdk-action-cfgs]:https://github.com/operator-framework/operator-sdk/tree/master/.github/workflows
[readme-platforms]:https://github.com/operator-framework/operator-sdk/tree/master/README.md#platforms
[deploy-workflow]:https://github.com/operator-framework/operator-sdk/tree/master/.github/workflows/deploy.yml
