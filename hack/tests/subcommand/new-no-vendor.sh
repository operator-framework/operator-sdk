#!/usr/bin/env bash

set -ex

source hack/lib/test_lib.sh

ROOTDIR="$(pwd)"
GOTMP="$(mktemp -d -p "$GOPATH/src")"
trap_add "rm -rf $GOTMP" EXIT

export GO111MODULE=on

pushd "$GOTMP"
operator-sdk new memcached-operator --skip-validation

edit_replace_modfile go.mod "$ROOTDIR"

# Esnure dependencies build correctly.
go build ./...
