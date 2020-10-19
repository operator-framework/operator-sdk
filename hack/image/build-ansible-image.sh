#!/usr/bin/env bash

set -eux

source hack/lib/test_lib.sh
source hack/lib/image_lib.sh

ROOTDIR="$(pwd)"
TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT

# build the base image
pushd $TMPDIR
cp $ROOTDIR/build/ansible-operator .
docker build -f $ROOTDIR/hack/image/ansible/Dockerfile -t $1 .
popd
