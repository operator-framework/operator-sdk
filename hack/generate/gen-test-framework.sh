#!/usr/bin/env bash

#############################
# Make sure that test/test-framework is updated according to the current source code
#############################

set -ex

source hack/lib/test_lib.sh

# Define vars
DOCKERFILE="build/Dockerfile"

# Go inside of the mock data test project
cd test/test-framework

# Run gen commands
../../build/operator-sdk generate k8s
../../build/operator-sdk generate openapi

# TODO(camilamacedo86): remove this when the openapi gen be set to false and it no longer is generated
# The following file is gen by openapi but it has not been committed in order to allow we clone and call the test locally in any path.
trap_add 'rm pkg/apis/cache/v1alpha1/zz_generated.openapi.go' EXIT
