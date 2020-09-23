#!/usr/bin/env bash

source hack/lib/common.sh
source hack/lib/test_lib.sh
source hack/lib/image_lib.sh

set -eu

header_text "Running tests to check ansible molecule"

ROOTDIR="$(pwd)"
TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT
pip3 install --user pyasn1==0.4.7 pyasn1-modules==0.2.6 idna==2.8 ipaddress==1.0.22
pip3 install --user molecule==3.0.2
pip3 install --user ansible-lint yamllint
pip3 install --user docker==4.2.2 openshift jmespath
ansible-galaxy collection install community.kubernetes

setup_envs $tmp_sdk_root

pushd "$TMPDIR"

header_text "Creating memcached-operator"
mkdir memcached-operator
pushd memcached-operator
operator-sdk init --plugins ansible.sdk.operatorframework.io/v1 \
  --domain example.com \
  --group ansible \
  --version v1alpha1 \
  --kind Memcached \
  --generate-playbook \
  --generate-role
header_text "Replacing operator contents"
cp "$ROOTDIR/test/ansible-memcached/tasks.yml" roles/memcached/tasks/main.yml
cp "$ROOTDIR/test/ansible-memcached/defaults.yml" roles/memcached/defaults/main.yml
cp "$ROOTDIR/test/ansible-memcached/memcached_test.yml"  molecule/default/tasks/memcached_test.yml
cp -a "$ROOTDIR/test/ansible-memcached/memfin" roles/
cp -a "$ROOTDIR/test/ansible-memcached/secret" roles/
marker=$(tail -n1 watches.yaml)
sed -i'.bak' -e '$ d' watches.yaml;rm -f watches.yaml.bak
cat "$ROOTDIR/test/ansible-memcached/watches-finalizer.yaml" >> watches.yaml
header_text "Append v1 kind to watches to test watching already registered GVK"
cat "$ROOTDIR/test/ansible-memcached/watches-v1-kind.yaml" >> watches.yaml
echo $marker >> watches.yaml
sed -i'.bak' -e '/- secrets/a \ \ \ \ \ \ - services' config/rbac/role.yaml; rm -f config/rbac/role.yaml.bak

header_text "Test in kind"
sed -i".bak" -E -e 's/(FROM quay.io\/operator-framework\/ansible-operator)(:.*)?/\1:dev/g' Dockerfile; rm -f Dockerfile.bak
OPERATORDIR="$(pwd)"
make kustomize
if [ -f ./bin/kustomize ] ; then
  KUSTOMIZE="$(realpath ./bin/kustomize)"
else
  KUSTOMIZE="$(which kustomize)"
fi
KUSTOMIZE_PATH=${KUSTOMIZE} TEST_OPERATOR_NAMESPACE=default molecule test -s kind

popd
popd
KUSTOMIZE_PATH=${KUSTOMIZE}
header_text "Test Ansible Molecule scenarios"
pushd "${ROOTDIR}/test/ansible"
DEST_IMAGE="quay.io/example/ansible-test-operator:v0.0.1"
sed -i".bak" -E -e 's/(FROM quay.io\/operator-framework\/ansible-operator)(:.*)?/\1:dev/g' build/Dockerfile; rm -f build/Dockerfile.bak
docker build -f build/Dockerfile -t "$DEST_IMAGE" --no-cache .
load_image_if_kind "$DEST_IMAGE"
OPERATOR_PULL_POLICY=Never OPERATOR_IMAGE=${DEST_IMAGE} TEST_CLUSTER_PORT=24443 TEST_OPERATOR_NAMESPACE=osdk-test molecule test --all

popd
