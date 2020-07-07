#!/usr/bin/env bash

# remove running containers on exit
function cleanup() {
  kind delete cluster
}

set -o errexit
set -o nounset
set -o pipefail

source ./hack/lib/common.sh
source ./hack/lib/test_lib.sh

test_dir=./test
tests=$test_dir/e2e-helm-new

export TRACE=1
export GO111MODULE=on

: ${K8S_VERSION:?"must be set"}

prepare_staging_dir $tmp_sdk_root
fetch_tools $tmp_sdk_root

header_text "###"
header_text "### These envtest environment variables are required for the default unit tests"
header_text "### scaffolded in the test operator project. No e2e tests currently use envtest."
header_text "###"

setup_envs $tmp_sdk_root
build_sdk $tmp_sdk_root
setup_kustomize $tmp_sdk_root

header_text "### Create a cluster of version $K8S_VERSION."

kind create cluster -v 4 --retain --wait=1m \
  --config $test_dir/kind-config.yaml \
  --image=kindest/node:$K8S_VERSION

kind export kubeconfig

kubectl cluster-info

header_text "###"
header_text "### Building Helm Image"
header_text "###"
make image-build-helm

header_text "###"
header_text "### Loading kubebuilder images"
header_text "###"
docker pull gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0
kind load docker-image gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0

trap_add cleanup EXIT

go test $tests -v -ginkgo.v
