#!/usr/bin/env bash

set -eux

source hack/lib/image_lib.sh

export OSDK_INTEGRATION_IMAGE="quay.io/example/memcached-operator:v0.0.1"

# Build the operator image.
pushd test/test-framework
operator-sdk build "$OSDK_INTEGRATION_IMAGE"
# If using a kind cluster, load the image into all nodes.
load_image_if_kind "$OSDK_INTEGRATION_IMAGE"
popd

# Install OLM on the cluster if not installed.
is_installed=0
if ! operator-sdk alpha olm status > /dev/null 2>&1; then
  operator-sdk alpha olm install
  is_installed=1
fi

# Integration tests will use default loading rules for the kubeconfig if
# KUBECONFIG is not set.
go test -v ./test/integration

# Uninstall OLM if it was installed for test purposes.
if eval "(( $is_installed ))"; then
  operator-sdk alpha olm uninstall
fi
