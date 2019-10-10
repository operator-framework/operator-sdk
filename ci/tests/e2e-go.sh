#!/usr/bin/env bash
set -ex

make install
# this configures the image correctly for memcached-operator
component="memcached-operator"
eval IMAGE=$IMAGE_FORMAT
set -- "--image-name=$IMAGE --local-image=false"
source ./hack/tests/scaffolding/e2e-go-scaffold.sh

pushd $BASEPROJECTDIR/memcached-operator
operator-sdk test local ./test/e2e --verbose
popd

go test ./test/e2e/... -root=. -globalMan=testdata/empty.yaml
