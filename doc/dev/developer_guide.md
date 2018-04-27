# Developer guide

This document explains how to setup your dev environment.

## Download Operator SDK

Go to https://github.com/operator-framework/operator-sdk and follow the [fork guide][fork_guide] to fork, clone, and setup the local operator-sdk repository.

## Vendor dependencies

We use [dep](https://github.com/golang/dep) to manage dependencies.
Run the following in the project root directory to update the vendored dependencies:

```sh
$ cd $GOPATH/src/github.com/operator-framework/operator-sdk
$ dep ensure 
```

## Build the Operator SDK CLI

Requirement:
- Go 1.9+

Build the Operator SDK CLI `operator-sdk` binary:

```sh
# TODO: replace this with the ./build script.
$ go install github.com/operator-framework/operator-sdk/commands/operator-sdk 
```

## Testing

Run unit tests:

```sh
TODO: use ./test script
```

[fork_guide]:https://help.github.com/articles/fork-a-repo/