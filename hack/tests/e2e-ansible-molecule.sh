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

header_text "Creating molecule sample"
go run ./hack/generate/samples/molecule/generate.go --path=$TMPDIR

pushd "$TMPDIR"
popd
cd $TMPDIR/memcached-molecule-operator

header_text "Test Kind"
make kustomize
if [ -f ./bin/kustomize ] ; then
  KUSTOMIZE="$(realpath ./bin/kustomize)"
else
  KUSTOMIZE="$(which kustomize)"
fi
KUSTOMIZE_PATH=${KUSTOMIZE} TEST_OPERATOR_NAMESPACE=default molecule test -s kind

rm -rf $KUSTOMIZE
cd $TMPDIR/
rm -rf memcached-molecule-operator

header_text "Test Ansible Molecule scenarios"
pushd "${ROOTDIR}/test/ansible"

make kustomize
if [ -f ./bin/kustomize ] ; then
  KUSTOMIZE="$(realpath ./bin/kustomize)"
else
  KUSTOMIZE="$(which kustomize)"
fi

DEST_IMAGE="quay.io/example/advanced-molecule-operator:v0.0.1"
sed -i".bak" -E -e 's/(FROM quay.io\/operator-framework\/ansible-operator)(:.*)?/\1:dev/g' Dockerfile; rm -f Dockerfile.bak
docker build -t "$DEST_IMAGE" --no-cache .
load_image_if_kind "$DEST_IMAGE"
KUSTOMIZE_PATH=$KUSTOMIZE OPERATOR_PULL_POLICY=Never OPERATOR_IMAGE=${DEST_IMAGE} TEST_OPERATOR_NAMESPACE=osdk-test molecule test
