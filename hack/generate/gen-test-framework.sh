#!/usr/bin/env bash

#############################
# Make sure that test/test-framework is updated according to the current source code
#############################

set -ex

source hack/lib/test_lib.sh

# Go inside of the mock data test project
cd test/test-framework

# Create temporary modfile for generators.
trap_add 'rm -f go.mod go.sum' EXIT
../../build/operator-sdk print-deps > go.mod
echo -e "\nreplace github.com/operator-framework/operator-sdk => ../../" >> go.mod
go build ./...

# Run gen commands
../../build/operator-sdk generate k8s
# TODO(camilamacedo86): remove this when the openapi gen be set to false and it no longer is generated
# The following file is gen by openapi but it has not been committed in order to allow we clone and call the test locally in any path.
trap_add 'rm -f pkg/apis/cache/v1alpha1/zz_generated.openapi.go' EXIT
../../build/operator-sdk generate openapi
