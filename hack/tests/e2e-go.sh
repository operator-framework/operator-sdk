#!/usr/bin/env bash
set -ex

source hack/lib/image_lib.sh

source hack/tests/scaffolding/e2e-go-scaffold.sh

pushd $BASEPROJECTDIR/memcached-operator
operator-sdk build $IMAGE_NAME
# If using a kind cluster, load the image into all nodes.
load_image_if_kind "$IMAGE_NAME"

operator-sdk test local ./test/e2e
popd

go test ./test/e2e/... -root=. -globalMan=testdata/empty.yaml
