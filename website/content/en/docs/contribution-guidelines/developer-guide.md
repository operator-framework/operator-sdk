---
title: Development Environment
weight: 30
---

This document explains how to setup your dev environment.

## Prerequisites
- [git][git-tool]
- [go][go-tool] version v1.13+

## Download Operator SDK

Go to the [Operator SDK repo][repo-sdk] and follow the [fork guide][fork-guide] to fork, clone, and setup the local operator-sdk repository.

## Build the Operator SDK CLI

Build the Operator SDK CLI `operator-sdk` binary:

```sh
$ make install
```

Then, now you are able to test and use the operator-sdk build using the source code.

## Testing

The SDK includes many tests that are run as part of CI.
To build the binary and run all tests (assuming you have a correctly configured environment),
you can simple run:

```sh
$ make test-ci
```

If you simply want to run the unit tests, you can run:

```sh
$ make test
```

For more information on running testing and correctly configuring your environment,
refer to the [`Running the Tests Locally`][running-the-tests] document. To incorporate code changes in your development environment see the [`Testing changes Locally`][testing-changes-locally] document.

To run the lint checks done in the CI locally, run:

```sh
$ make lint
```

**NOTE** Note that for it is required to install `golangci-lint` locally. For more info see its [doc](https://github.com/golangci/golangci-lint#install)

## How the operator-sdk binaries are built

In the release process, the script [.release.sh][release-sh] is executed and it will call the [makefile][makefile] target [make release](https://github.com/operator-framework/operator-sdk/blob/master/Makefile#L113). To know more about the release process, see the [doc][release-doc] also see [operator-sdk releases](https://github.com/operator-framework/operator-sdk/releases).

**NOTE** The Deploy stage (configured in [.travis.yml ][travis]) builds also execute the same [makefile][makefile] targets. This stage is executed against the master branch when a Pull Request is merged.

## How to test the build of operator-sdk binaries

Follow these steps to execute the Travis `Deploy` stage against your branch to demonstrate that the merge build will complete as expected.

- Enable the Travis in your fork repository. For more info see [`To get started with Travis CI using GitHub`](https://docs.travis-ci.com/user/tutorial/#to-get-started-with-travis-ci-using-github) 
- Create image repos in quay (or another registry that supports multi-arch images) for ansible, helm, and scorecard proxy. For each image type, you need repos for the manifest list and one for each architecture (e.g. `ansible-operator`, `ansible-operator-amd64`, `ansible-operator-s390x`, etc.)

**NOTE** Be sure to make each repository public.

- Set the following environment variables in the Travis settings for your fork:

    - `ANSIBLE_IMAGE` docker image name (e.g. `quay.io/joelanford/ansible-operator`)
    - `HELM_IMAGE` same as above, but for helm
    - `SCORECARD_PROXY_IMAGE` same as above, but for scorecard proxy
    - `DOCKER_USERNAME` credentials for your repo
    - `DOCKER_PASSWORD` credentials for your repo
    - `DOCKER_CLI_EXPERIMENTAL`  set to `enabled` 
    - `COVERALLS_TOKEN`  token to integrate the project with `https://coveralls.io/`. So, enable your fork in `https://coveralls.io/` and generate a token to allow it. 

- Make a commit with `[travis deploy]` in the commit message on the branch with the changes.
- Check the travis build for your branch in your fork (not the PR build in the operator-sdk repo, since we don't allow PRs to build images in the `operator-framework` quay repo.)

**NOTE** Post a link in the Pull Request to the Travis build page showing successful `Deploy` and `Deploy multi-arch manifest lists` stages with your changes.

See the project [README][sdk-readme] for more details.

[git-tool]:https://git-scm.com/downloads
[go-tool]:https://golang.org/dl/
[repo-sdk]:https://github.com/operator-framework/operator-sdk
[fork-guide]:https://help.github.com/en/articles/fork-a-repo
[docker-tool]:https://docs.docker.com/install/
[kubectl-tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[sdk-readme]: https://github.com/operator-framework/operator-sdk/blob/master/README.md
[running-the-tests]: ../testing/running-the-tests
[testing-changes-locally]: ../local-changes 
[makefile]: https://github.com/operator-framework/operator-sdk/blob/master/Makefile
[travis]: https://github.com/operator-framework/operator-sdk/blob/master/.travis.yml
[release-sh]: https://github.com/operator-framework/operator-sdk/blob/master/release.sh
[release-doc]: ../release
