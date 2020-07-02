#!/usr/bin/env bash

set -eux

source hack/lib/test_lib.sh
source hack/lib/image_lib.sh

ROOTDIR="$(pwd)"
TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT
BASEIMAGEDIR="$TMPDIR/helm-operator"
mkdir -p "$BASEIMAGEDIR"
go build -o $BASEIMAGEDIR/scaffold-helm-image ./hack/image/helm/scaffold-helm-image.go

# build binary for specific target platform (for purposes of base image only)
env GOOS=linux GOARCH=amd64 go build -o $BASEIMAGEDIR/base-image ./hack/image/helm/baseimage/baseimage.go

# build operator binary and base image
pushd "$BASEIMAGEDIR"
./scaffold-helm-image

mkdir -p build/_output/bin/
mv $BASEIMAGEDIR/base-image build/_output/bin
operator-sdk build $1
# If using a kind cluster, load the image into all nodes.
load_image_if_kind "$1"
popd
