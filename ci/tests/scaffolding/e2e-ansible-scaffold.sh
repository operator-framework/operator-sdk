#!/usr/bin/env bash

source hack/lib/test_lib.sh

set -eux

ROOTDIR="$(pwd)"

mkdir -p /ansible
cd /ansible

# create and build the operator
operator-sdk new memcached-operator --api-version=ansible.example.com/v1alpha1 --kind=Memcached --type=ansible
cp "$ROOTDIR/test/ansible-memcached/tasks.yml" memcached-operator/roles/memcached/tasks/main.yml
cp "$ROOTDIR/test/ansible-memcached/defaults.yml" memcached-operator/roles/memcached/defaults/main.yml
cp -a "$ROOTDIR/test/ansible-memcached/memfin" memcached-operator/roles/
cat "$ROOTDIR/test/ansible-memcached/watches-finalizer.yaml" >> memcached-operator/watches.yaml
# Append Foo kind to watches to test watching multiple Kinds
cat "$ROOTDIR/test/ansible-memcached/watches-foo-kind.yaml" >> memcached-operator/watches.yaml
