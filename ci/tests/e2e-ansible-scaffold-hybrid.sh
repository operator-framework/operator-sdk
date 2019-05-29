#!/usr/bin/env bash

source hack/lib/test_lib.sh

set -eux

ROOTDIR="$(pwd)"
ANSIBLEDIR="/go/src/github.com/ansible-op"

mkdir -p $ANSIBLEDIR
cd $ANSIBLEDIR

# create and build the operator
operator-sdk new memcached-operator --api-version=ansible.example.com/v1alpha1 --kind=Memcached --type=ansible
cp "$ROOTDIR/test/ansible-memcached/tasks.yml" memcached-operator/roles/memcached/tasks/main.yml
cp "$ROOTDIR/test/ansible-memcached/defaults.yml" memcached-operator/roles/memcached/defaults/main.yml
cp -a "$ROOTDIR/test/ansible-memcached/memfin" memcached-operator/roles/
cat "$ROOTDIR/test/ansible-memcached/watches-finalizer.yaml" >> memcached-operator/watches.yaml
# Append Foo kind to watches to test watching multiple Kinds
cat "$ROOTDIR/test/ansible-memcached/watches-foo-kind.yaml" >> memcached-operator/watches.yaml

pushd memcached-operator
# Add a second Kind to test watching multiple GVKs
operator-sdk add crd --kind=Foo --api-version=ansible.example.com/v1alpha1

export GO111MODULE=on
operator-sdk migrate

if [[ ! -e build/Dockerfile.sdkold ]];
then
    echo FAIL the old Dockerfile should have been renamed to Dockerfile.sdkold
    exit 1
fi

# Run `go build ./...` to pull down the deps specified by the scaffolded
# `go.mod` file and verify dependencies build correctly.
go build ./...

# Use the local operator-sdk directory as the repo. To make the go toolchain
# happy, the directory needs a `go.mod` file that specifies the module name,
# so we need this temporary hack until we update the SDK repo itself to use
# go modules.
echo "module github.com/operator-framework/operator-sdk" > $ROOTDIR/go.mod
go mod edit -replace=github.com/operator-framework/operator-sdk=$ROOTDIR
go build ./...

popd
