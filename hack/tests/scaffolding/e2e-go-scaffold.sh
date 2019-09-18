#!/usr/bin/env bash

set -ex

source hack/lib/test_lib.sh

ROOTDIR="$(pwd)"
BASEPROJECTDIR="$(mktemp -d)"
IMAGE_NAME="quay.io/example/memcached-operator:v0.0.1"

# change IMAGE_NAME in case it is redefined by a flag
# parse "--image-name value" format
ORIG_ARGS=("$@")
while [[ $# -gt 0 ]]
do
key="$1"
case $key in
    -image-name|--image-name) # "--image-name value" format
    IMAGE_NAME="$2"
    shift
    ;;
    -image-name=*|--image-name=*) # "--image-name=value" format
    IMAGE_NAME="${key#*=}"
    shift
    ;;
    *)    # different arg/flag
    shift
    ;;
esac
done

set -- "${ORIG_ARGS[@]}"

go build -o $BASEPROJECTDIR/scaffold-memcached $ROOTDIR/hack/tests/scaffolding/scaffold-memcached.go

pushd "$BASEPROJECTDIR"
./scaffold-memcached --local-repo $ROOTDIR --image-name=$IMAGE_NAME --local-image $@
popd
