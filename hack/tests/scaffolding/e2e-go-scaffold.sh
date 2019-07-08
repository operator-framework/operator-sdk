#!/usr/bin/env bash

set -ex

source hack/lib/test_lib.sh

ROOTDIR="$(pwd)"
# TODO: remove once PR 1566 is merged
trap_add 'rm -f $ROOTDIR/go.mod' EXIT
BASEPROJECTDIR="$(mktemp -d)"
IMAGE_NAME="quay.io/example/memcached-operator:v0.0.1"

go build -o $BASEPROJECTDIR/scaffold-memcached $ROOTDIR/hack/tests/scaffolding/scaffold-memcached.go

pushd "$BASEPROJECTDIR"
./scaffold-memcached --local-repo $ROOTDIR --image-name=$IMAGE_NAME --local-image
popd
