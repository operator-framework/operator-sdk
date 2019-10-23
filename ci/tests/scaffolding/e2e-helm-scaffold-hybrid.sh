#!/usr/bin/env bash

source hack/lib/test_lib.sh

set -eux

ROOTDIR="$(pwd)"
HELMDIR="/go/src/github.com/helm-op"
mkdir -p $HELMDIR

# create and build the operator
pushd "$HELMDIR"
operator-sdk new nginx-operator --api-version=helm.example.com/v1alpha1 --kind=Nginx --type=helm

pushd nginx-operator
export GO111MODULE=on
operator-sdk migrate

if [[ ! -e build/Dockerfile.sdkold ]];
then
    echo FAIL the old Dockerfile should have been renamed to Dockerfile.sdkold
    exit 1
fi

add_go_mod_replace "github.com/operator-framework/operator-sdk" "$ROOTDIR"
# Build the project to resolve dependency versions in the modfile.
go build ./...

WD="$(dirname "$(pwd)")"
go build -gcflags "all=-trimpath=${WD}" -asmflags "all=-trimpath=${WD}" -o /nginx-operator github.com/helm-op/nginx-operator/cmd/manager
popd
popd
mv $HELMDIR /helm
