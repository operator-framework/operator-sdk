#!/usr/bin/env bash

source hack/lib/test_lib.sh

set -eux

ROOTDIR="$(pwd)"
TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT
pip3 install --user pyasn1==0.4.7 pyasn1-modules==0.2.6 idna==2.8 ipaddress==1.0.22
pip3 install --user molecule==2.22
pip3 install --user docker openshift jmespath

deploy_prereqs() {
    kubectl create -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl create -f "$OPERATORDIR/deploy/crds/ansible.example.com_memcacheds_crd.yaml"
}

remove_prereqs() {
    kubectl delete --wait=true --ignore-not-found=true --timeout=60s -f "$OPERATORDIR/deploy/crds/ansible.example.com_memcacheds_crd.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/role.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/role_binding.yaml"
}

pushd "$TMPDIR"
operator-sdk new memcached-operator \
  --api-version=ansible.example.com/v1alpha1 \
  --kind=Memcached \
  --type=ansible \
  --generate-playbook
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
# Use the following sed command to check it on macOsX.
# More info: https://www.mkyong.com/mac/sed-command-hits-undefined-label-error-on-mac-os-x/
# sed -i "" 's|\(FROM quay.io/operator-framework/ansible-operator\)\(:.*\)\?|\1:dev|g' build/Dockerfile
# The following code is the default used (Not valid for MacOSX)
sed -i 's|\(FROM quay.io/operator-framework/ansible-operator\)\(:.*\)\?|\1:dev|g' build/Dockerfile
OPERATORDIR="$(pwd)"
TEST_CLUSTER_PORT=24443 operator-sdk test local --namespace default

remove_prereqs

popd
popd

pushd "${ROOTDIR}/test/ansible-inventory"
# Use the following sed command to check it on macOsX.
# More info: https://www.mkyong.com/mac/sed-command-hits-undefined-label-error-on-mac-os-x/
# sed -i "" 's|\(FROM quay.io/operator-framework/ansible-operator\)\(:.*\)\?|\1:dev|g' build/Dockerfile
# The following code is the default used (Not valid for MacOSX)
sed -i 's|\(FROM quay.io/operator-framework/ansible-operator\)\(:.*\)\?|\1:dev|g' build/Dockerfile
TEST_CLUSTER_PORT=24443 operator-sdk test local --namespace default

popd
