# Developer guide

This document explains how to setup your dev environment.

## Prerequisites
- [git][git-tool]
- [go][go-tool] version v1.12+

## Download Operator SDK

Go to the [Operator SDK repo][repo-sdk] and follow the [fork guide][fork-guide] to fork, clone, and setup the local operator-sdk repository.

## Vendor dependencies

Run the following in the project root directory to update the vendored dependencies:

```sh
$ cd $GOPATH/src/github.com/operator-framework/operator-sdk
$ make dep
```

## Build the Operator SDK CLI

Build the Operator SDK CLI `operator-sdk` binary:

```sh
$ make install
```

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
refer to the [`Running the Tests Locally`][running-the-tests] document.

See the project [README][sdk-readme] for more details.

[git-tool]:https://git-scm.com/downloads
[go-tool]:https://golang.org/dl/
[repo-sdk]:https://github.com/operator-framework/operator-sdk
[fork-guide]:https://help.github.com/en/articles/fork-a-repo
[docker-tool]:https://docs.docker.com/install/
[kubectl-tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[sdk-readme]:../../README.md
[running-the-tests]: ./testing/running-the-tests.md
