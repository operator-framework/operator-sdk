#!/usr/bin/env bash

source hack/lib/common.sh

# load_image_if_kind <image tag>
#
# load_image_if_kind loads an image into all nodes in a kind cluster.
#
function load_image_if_kind() {
  local cluster=${KIND_CLUSTER:-kind}
  if [[ "$(kubectl config current-context)" == "kind-${cluster}" ]]; then
    kind load docker-image --name "${cluster}" "$1"
  fi
}

set -eu

header_text "Running ansible molecule tests in a python3 virtual environment"

# Set up a python3.8 virtual environment.
ENVDIR="$(mktemp -d)"
trap_add "set +u; deactivate; set -u; rm -rf $ENVDIR" EXIT
python3 -m venv "$ENVDIR"
set +u; source "${ENVDIR}/bin/activate"; set -u

# Install dependencies.
TMPDIR="$(mktemp -d)"
trap_add "rm -rf $TMPDIR" EXIT
pip3 install pyasn1==0.4.7 pyasn1-modules==0.2.6 idna==2.8 ipaddress==1.0.23
pip3 install cryptography==3.3.2 molecule==3.0.2
pip3 install ansible-lint yamllint
pip3 install docker==4.2.2 openshift==0.12.1 jmespath
ansible-galaxy collection install 'kubernetes.core:==2.2.0'
ansible-galaxy collection install 'operator_sdk.util:==0.4.0'

header_text "Copying molecule testdata scenarios"
ROOTDIR="$(pwd)"
cp -r $ROOTDIR/testdata/ansible/memcached-molecule-operator/ $TMPDIR/memcached-molecule-operator
cp -r $ROOTDIR/testdata/ansible/advanced-molecule-operator/ $TMPDIR/advanced-molecule-operator

pushd $TMPDIR/memcached-molecule-operator

header_text "Running Kind test with memcached-molecule-operator"
make kustomize
if [ -f ./bin/kustomize ] ; then
  KUSTOMIZE="$(realpath ./bin/kustomize)"
else
  KUSTOMIZE="$(which kustomize)"
fi
KUSTOMIZE_PATH=${KUSTOMIZE} TEST_OPERATOR_NAMESPACE=default molecule test -s kind
popd

header_text "Running Default test with advanced-molecule-operator"

make test-e2e-setup
pushd $TMPDIR/advanced-molecule-operator

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
popd
