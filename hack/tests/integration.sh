#!/usr/bin/env bash

set -eu

source hack/lib/common.sh
source hack/lib/test_lib.sh
source hack/lib/image_lib.sh

TMPDIR="$(mktemp -d -p /tmp memcached-operator-XXXX)"
trap_add 'rm -rf $TMPDIR' EXIT
pushd "$TMPDIR"

###########################################################################
### DO NOT UNCOMMENT THESE LINES UNLESS YOU KNOW WHAT YOU'RE DOING !!!! ###
###                                                                     ###
### They cause the integration image not to be loaded into kind in      ###
### TravisCI.                                                           ###
###                                                                     ###
###########################################################################
###
###    #prepare_staging_dir $tmp_sdk_root
###    #fetch_envtest_tools $tmp_sdk_root
###
###########################################################################
setup_envs $tmp_sdk_root

header_text "Initializing test project"

# Initialize a basic memcached-operator project
operator-sdk init --repo github.com/example/memcached-operator --domain example.com --fetch-deps=false
operator-sdk create api --group cache --version v1alpha1 --kind Memcached --controller --resource
sed -i 's@Foo string `json:"foo,omitempty"`@// +optional\
        Count int `json:"count,omitempty"`@' api/v1alpha1/memcached_types.go

# Build the operator's image.
export OSDK_INTEGRATION_IMAGE="quay.io/example/memcached-operator:integration"
make docker-build IMG="$OSDK_INTEGRATION_IMAGE"
load_image_if_kind "$OSDK_INTEGRATION_IMAGE"

popd

# Install OLM on the cluster if not installed.
olm_latest_exists=0
if ! operator-sdk olm status > /dev/null 2>&1; then
  operator-sdk olm install
  olm_latest_exists=1
fi

docker pull gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0
load_image_if_kind gcr.io/kubebuilder/kube-rbac-proxy:v0.5.0

header_text "Running integration tests"

# Integration tests will use default loading rules for the kubeconfig if KUBECONFIG is not set.
go test -v ./test/integration

header_text "Integration tests succeeded"

# Uninstall OLM if it was installed for test purposes.
if eval "(( $olm_latest_exists ))"; then
  operator-sdk olm uninstall
fi
