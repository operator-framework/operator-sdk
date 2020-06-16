#!/usr/bin/env bash

set -ex

source hack/lib/test_lib.sh

function download_old_sdk_binary() {
    VERSION="v0.18.1"
    echo "Getting SDK $VERSION release binary..."
    BIN_URL=""
    OS=$(go env GOOS)
    if [ "$OS" = "darwin" ]; then
        BIN_URL="https://github.com/operator-framework/operator-sdk/releases/download/$VERSION/operator-sdk-$VERSION-x86_64-apple-darwin"
    elif [ "$OS" = "linux" ]; then
        BIN_URL="https://github.com/operator-framework/operator-sdk/releases/download/$VERSION/operator-sdk-$VERSION-x86_64-linux-gnu"
    else
        echo "Failed to get SDK $VERSION release binary for $OS"
        exit 1
    fi

    curl -o operator-sdk-old -L $BIN_URL
    chmod +x ./operator-sdk-old
}

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
# scaffold-memcached uses "operator-sdk-old new ..." to scaffold the legacy project
# since "new" is not present in the new CLI
download_old_sdk_binary
./scaffold-memcached --local-repo $ROOTDIR --image-name=$IMAGE_NAME --local-image $@
popd
