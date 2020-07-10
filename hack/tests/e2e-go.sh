#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

source ./hack/lib/test_lib.sh
source ./hack/lib/image_lib.sh

test_dir=./test
tests=$test_dir/e2e

export TRACE=1
export GO111MODULE=on

prepare_staging_dir $tmp_sdk_root
fetch_tools $tmp_sdk_root
# These envtest environment variables are required for the default unit tests
# scaffolded in the test operator project. No e2e tests currently use envtest.
setup_envs $tmp_sdk_root
build_sdk $tmp_sdk_root

kubectl cluster-info

docker pull gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0
load_image_if_kind gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0

go test -v $tests
