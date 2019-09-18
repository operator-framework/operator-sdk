#!/usr/bin/env bash
set -ex

# Scaffold project (image name and local image args don't matter at this stage as those only affect the operator manifest)
source ./hack/tests/scaffolding/e2e-go-scaffold.sh

pushd $BASEPROJECTDIR/memcached-operator
go build -gcflags "all=-trimpath=${GOPATH}" -asmflags "all=-trimpath=${GOPATH}" -o /memcached-operator $BASEPROJECTDIR/memcached-operator/cmd/manager
popd
