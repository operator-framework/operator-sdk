#!/usr/bin/env bash

set -ex

source hack/lib/test_lib.sh

ROOTDIR="$(pwd)"
trap_add 'rm $ROOTDIR/go.mod || true' EXIT
GOTMP="$(mktemp -d)"
trap_add 'rm -rf $GOTMP' EXIT
BASEPROJECTDIR="/tmp/go-e2e-scaffold"
IMAGE_NAME="quay.io/example/memcached-operator:v0.0.1"

rm -rf $BASEPROJECTDIR
mkdir -p $BASEPROJECTDIR

pushd "$BASEPROJECTDIR"
go run "$ROOTDIR/hack/tests/scaffolding/scaffold-memcached.go" --local-repo $ROOTDIR --image-name=$IMAGE_NAME --local-image
popd
