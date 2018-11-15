# Developer guide

This document explains how to setup your dev environment.

## Prerequisites
- [dep][dep_tool] version v0.5.0+
- [git][git_tool]
- [go][go_tool] version v1.10+

## Download Operator SDK

Go to the [Operator SDK repo][repo_sdk] and follow the [fork guide][fork_guide] to fork, clone, and setup the local operator-sdk repository.

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

Run unit tests:

```sh
$ make test/unit
```

Run all tests:

```sh
$ make test
```

**Note:** running all tests requires:
- [docker][docker_tool] version 17.03+
- [kubectl][kubectl_tool] version v1.11.0+
- Access to a kubernetes v.1.11.0+ cluster

See the project [README][sdk_readme] for more details.

[dep_tool]:https://golang.github.io/dep/docs/installation.html
[git_tool]:https://git-scm.com/downloads
[go_tool]:https://golang.org/dl/
[repo_sdk]:https://github.com/operator-framework/operator-sdk
[fork_guide]:https://help.github.com/articles/fork-a-repo/
[docker_tool]:https://docs.docker.com/install/
[kubectl_tool]:https://kubernetes.io/docs/tasks/tools/install-kubectl/
[sdk_readme]:../../README.md
