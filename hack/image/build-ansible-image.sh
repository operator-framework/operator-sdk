#!/usr/bin/env bash

set -eux

source hack/lib/test_lib.sh

ROOTDIR="$(pwd)"
GOTMP="$(mktemp -d)"
trap_add 'rm -rf $GOTMP' EXIT
BASEIMAGEDIR="$GOTMP/ansible-operator"
mkdir -p "$BASEIMAGEDIR"
go build -o $BASEIMAGEDIR/scaffold-ansible-image ./hack/image/ansible/scaffold-ansible-image.go

# build operator binary and base image
pushd "$BASEIMAGEDIR"
./scaffold-ansible-image

mkdir -p build/_output/bin/
cp $ROOTDIR/build/operator-sdk-dev-x86_64-linux-gnu build/_output/bin/ansible-operator
operator-sdk build $1
popd
