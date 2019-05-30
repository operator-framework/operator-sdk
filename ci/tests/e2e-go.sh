#!/usr/bin/env bash
set -ex

make install
# this configures the image correctly for memcached-operator
component="memcached-operator"
eval IMAGE=$IMAGE_FORMAT
go test ./test/e2e/... -root=. -globalMan=testdata/empty.yaml -v -no-image-build -image $IMAGE $1
