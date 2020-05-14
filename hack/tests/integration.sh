#!/usr/bin/env bash

set -eu

source hack/lib/image_lib.sh

export OSDK_INTEGRATION_IMAGE="quay.io/example/memcached-operator:latest"

# Build the operator image.
pushd test/test-framework
operator-sdk build "$OSDK_INTEGRATION_IMAGE"
# If using a kind cluster, load the image into all nodes.
load_image_if_kind "$OSDK_INTEGRATION_IMAGE"
popd

# Install OLM on the cluster if not installed.
olm_latest_exists=0
if ! operator-sdk olm status > /dev/null 2>&1; then
  operator-sdk olm install
  olm_latest_exists=1
fi

# Integration tests will use default loading rules for the kubeconfig if
# KUBECONFIG is not set.
go test -v ./test/integration

# Uninstall OLM if it was installed for test purposes.
if eval "(( $olm_latest_exists ))"; then
  operator-sdk olm uninstall
fi

echo -e "\n=== Integration tests succeeded ===\n"
