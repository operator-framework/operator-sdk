#!/usr/bin/env bash

set -eux

# Prow CI setup.
component="memcached-operator"
eval IMAGE=$IMAGE_FORMAT
export OSDK_INTEGRATION_IMAGE="$IMAGE"

# Integration tests will use default loading rules for the kubeconfig if
# KUBECONFIG is not set.
# Assumes OLM is installed.
go test -v ./test/integration
