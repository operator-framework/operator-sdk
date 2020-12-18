#!/usr/bin/env bash

source hack/lib/common.sh
source hack/lib/image_lib.sh

set -eu

header_text "Running tests to check ansible molecule"

ROOTDIR="$(pwd)"
TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT
export PATH=${HOME}/.local/bin:${PATH}
pip3 install --user pyasn1==0.4.7 pyasn1-modules==0.2.6 idna==2.8 ipaddress==1.0.22
pip3 install --user molecule==3.0.2
pip3 install --user ansible-lint yamllint
pip3 install --user docker==4.2.2 openshift jmespath
ansible-galaxy collection install 'community.kubernetes:<1.0.0'

header_text "Copying molecule testdata scenarios"
cp -r $ROOTDIR/testdata/ansible/memcached-molecule-operator/ $TMPDIR/memcached-molecule-operator
cp -r $ROOTDIR/testdata/ansible/advanced-molecule-operator/ $TMPDIR/advanced-molecule-operator

pushd "$TMPDIR"
popd
cd $TMPDIR/memcached-molecule-operator

header_text "Running Kind test with memcached-molecule-operator/"
make kustomize
if [ -f ./bin/kustomize ] ; then
  KUSTOMIZE="$(realpath ./bin/kustomize)"
else
  KUSTOMIZE="$(which kustomize)"
fi
KUSTOMIZE_PATH=${KUSTOMIZE} TEST_OPERATOR_NAMESPACE=default molecule test -s kind


header_text "Running Default test with advanced-molecule-operator/"
cd $TMPDIR/advanced-molecule-operator

make kustomize
if [ -f ./bin/kustomize ] ; then
  KUSTOMIZE="$(realpath ./bin/kustomize)"
else
  KUSTOMIZE="$(which kustomize)"
fi

DEST_IMAGE="quay.io/example/advanced-molecule-operator:v0.0.1"
docker build -t "$DEST_IMAGE" --no-cache .
load_image_if_kind "$DEST_IMAGE"
KUSTOMIZE_PATH=$KUSTOMIZE OPERATOR_PULL_POLICY=Never OPERATOR_IMAGE=${DEST_IMAGE} TEST_OPERATOR_NAMESPACE=osdk-test molecule test
