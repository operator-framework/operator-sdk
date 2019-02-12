#!/usr/bin/env bash

source hack/lib/test_lib.sh

set -eux

ROOTDIR="$(pwd)"
GOTMP="$(mktemp -d -p $GOPATH/src)"
trap_add 'rm -rf $GOTMP' EXIT
# Needs to be from source until 2.20 comes out
pip install --user git+https://github.com/ansible/molecule.git
pip install --user docker openshift jmespath

deploy_prereqs() {
    kubectl create -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl create -f "$OPERATORDIR/deploy/crds/ansible_v1alpha1_memcached_crd.yaml"
}

remove_prereqs() {
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/role.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/crds/ansible_v1alpha1_memcached_crd.yaml"
}

pushd "$GOTMP"
operator-sdk new memcached-operator --api-version=ansible.example.com/v1alpha1 --kind=Memcached --type=ansible --generate-playbook
cp "$ROOTDIR/test/ansible-memcached/tasks.yml" memcached-operator/roles/memcached/tasks/main.yml
cp "$ROOTDIR/test/ansible-memcached/defaults.yml" memcached-operator/roles/memcached/defaults/main.yml
cp "$ROOTDIR/test/ansible-memcached/asserts.yml"  memcached-operator/molecule/default/asserts.yml
cp "$ROOTDIR/test/ansible-memcached/molecule.yml"  memcached-operator/molecule/test-local/molecule.yml
cp -a "$ROOTDIR/test/ansible-memcached/memfin" memcached-operator/roles/
cp -a "$ROOTDIR/test/ansible-memcached/secret" memcached-operator/roles/
cat "$ROOTDIR/test/ansible-memcached/watches-finalizer.yaml" >> memcached-operator/watches.yaml
cat "$ROOTDIR/test/ansible-memcached/prepare-test-image.yml" >> memcached-operator/molecule/test-local/prepare.yml
# Append v1 kind to watches to test watching already registered GVK
cat "$ROOTDIR/test/ansible-memcached/watches-v1-kind.yaml" >> memcached-operator/watches.yaml


# Test local
pushd memcached-operator
sed -i 's|\(FROM quay.io/operator-framework/ansible-operator\)\(:.*\)\?|\1:dev|g' build/Dockerfile
OPERATORDIR="$(pwd)"
TEST_CLUSTER_PORT=24443 operator-sdk test local --namespace default

# Test cluster
DEST_IMAGE="quay.io/example/memcached-operator:v0.0.2-test"
operator-sdk build --enable-tests "$DEST_IMAGE"
trap_add 'remove_prereqs' EXIT
deploy_prereqs
operator-sdk test cluster --image-pull-policy Never --namespace default --service-account memcached-operator ${DEST_IMAGE}

remove_prereqs

popd
popd

pushd "${ROOTDIR}/test/ansible-inventory"

sed -i 's|\(FROM quay.io/operator-framework/ansible-operator\)\(:.*\)\?|\1:dev|g' build/Dockerfile
TEST_CLUSTER_PORT=24443 operator-sdk test local --namespace default

popd
