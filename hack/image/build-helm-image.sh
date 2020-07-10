#!/usr/bin/env bash

set -eux

source hack/lib/test_lib.sh
source hack/lib/image_lib.sh

ROOTDIR="$(pwd)"
TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT

# build the base image
pushd $TMPDIR
cp $ROOTDIR/build/helm-operator-dev-linux-gnu .
docker build -f $ROOTDIR/hack/image/helm/Dockerfile -t $1 .

# If using a kind cluster, load the image into all nodes.
load_image_if_kind "$1"
popd
