#!/usr/bin/env bash

set -eux

source hack/lib/test_lib.sh
source hack/lib/image_lib.sh

ROOTDIR="$(pwd)"
TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT

# build scorecard test image
WD="$(dirname "$(pwd)")"
GOOS=linux CGO_ENABLED=0 \
  go build \
  -gcflags "all=-trimpath=${WD}" \
  -asmflags "all=-trimpath=${WD}" \
  -o $TMPDIR/scorecard-test \
  images/scorecard-test/cmd/test/main.go

# Build base image
pushd $TMPDIR
cp -r $ROOTDIR/images/scorecard-test/bin .
docker build -f $ROOTDIR/images/scorecard-test/Dockerfile -t $1 .
popd
