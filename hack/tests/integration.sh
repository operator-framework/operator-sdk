#!/usr/bin/env bash

set -eu

source hack/lib/common.sh
source hack/lib/image_lib.sh

TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT
pushd "$TMPDIR"
cd $TMPDIR
mkdir memcached-operator-XXXX
cd memcached-operator-XXXX

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

# todo: upgrade to use the 0.17.0 when the issue  https://github.com/operator-framework/operator-sdk/issues/4284 be solved
# Install OLM on the cluster if not installed.
olm_latest_exists=0
if ! operator-sdk olm status > /dev/null 2>&1; then
  operator-sdk olm install --version=0.15.1
  olm_latest_exists=1
fi

header_text "Running integration tests"

# Integration tests will use default loading rules for the kubeconfig if KUBECONFIG is not set.
go test -v ./test/integration

header_text "Integration tests succeeded"

# todo: upgrade to use the 0.17.0 when the issue  https://github.com/operator-framework/operator-sdk/issues/4284 be solved
# Uninstall OLM if it was installed for test purposes.
if eval "(( $olm_latest_exists ))"; then
  operator-sdk olm uninstall --version=0.15.1
fi
