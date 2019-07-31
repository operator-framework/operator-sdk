#!/usr/bin/env bash
set -ex

source hack/tests/scaffolding/e2e-go-scaffold.sh

pushd $BASEPROJECTDIR/memcached-operator
operator-sdk build $IMAGE_NAME

operator-sdk test local ./test/e2e
popd

go test ./test/e2e/... -root=. -globalMan=testdata/empty.yaml
