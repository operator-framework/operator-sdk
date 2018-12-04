#!/usr/bin/env bash

set -eux

source hack/lib/test_lib.sh

ROOTDIR="$(pwd)"
GOTMP="$(mktemp -d -p $GOPATH/src)"
trap_add 'rm -rf $GOTMP' EXIT
BASEIMAGEDIR="$GOTMP/ansible-operator"
mkdir -p "$BASEIMAGEDIR"

# build operator binary and base image
pushd "$BASEIMAGEDIR"
go run "$ROOTDIR/commands/ansible-operator-base/main.go"
dep ensure

# overwrite operator-sdk source with the latest source from the local checkout
pushd vendor/github.com/operator-framework/
rm -Rf operator-sdk/*
cp -a "$ROOTDIR"/{pkg,version,LICENSE} operator-sdk/
popd

operator-sdk build $1
popd
