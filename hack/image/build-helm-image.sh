#!/usr/bin/env bash

set -eux

source hack/lib/test_lib.sh

ROOTDIR="$(pwd)"
GOTMP="$(mktemp -d)"
trap_add 'rm -rf $GOTMP' EXIT
BASEIMAGEDIR="$GOTMP/helm-operator"
mkdir -p "$BASEIMAGEDIR"
go build -o $BASEIMAGEDIR/scaffold-helm-image ./hack/image/helm/scaffold-helm-image.go

# build operator binary and base image
pushd "$BASEIMAGEDIR"
./scaffold-helm-image

mkdir -p build/_output/bin/
cp $ROOTDIR/build/operator-sdk-dev-x86_64-linux-gnu build/_output/bin/helm-operator
operator-sdk build $1
popd
