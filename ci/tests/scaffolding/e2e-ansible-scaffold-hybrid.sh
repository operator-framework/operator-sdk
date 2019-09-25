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
export GOPROXY=https://proxy.golang.org
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
go build -gcflags "all=-trimpath=${WD}" -asmflags "all=-trimpath=${WD}" -o /memcached-operator github.com/ansible-op/memcached-operator/cmd/manager
popd
mv $ANSIBLEDIR /ansible
