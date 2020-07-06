#!/usr/bin/env bash

set -eux

source hack/lib/image_lib.sh

# build scorecard test kuttl image
WD="$(dirname "$(pwd)")"
GOOS=linux CGO_ENABLED=0 \
  go build \
  -gcflags "all=-trimpath=${WD}" \
  -asmflags "all=-trimpath=${WD}" \
  -o images/scorecard-test-kuttl/scorecard-test-kuttl \
  images/scorecard-test-kuttl/cmd/test-kuttl/main.go

# Build base image
pushd images/scorecard-test-kuttl
docker build -t "$1" .
# If using a kind cluster, load the image into all nodes.
load_image_if_kind "$1"
popd
