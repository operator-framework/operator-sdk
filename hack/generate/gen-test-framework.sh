#!/usr/bin/env bash

#############################
# Make sure that test/test-framework is updated according to the current source code
#############################

set -ex

source hack/lib/test_lib.sh

# Go inside of the mock data test project
cd test/test-framework

# Ensure test-framework is up-to-date with current Go project dependencies.
echo "$(../../build/operator-sdk print-deps)" > go.mod
sed -i".bak" -E -e "/github.com\/operator-framework\/operator-sdk .+/d" go.mod; rm -f go.mod.bak
echo -e "\nreplace github.com/operator-framework/operator-sdk => ../../" >> go.mod
go mod edit -require "github.com/operator-framework/operator-sdk@v0.0.0"
go build ./...
go mod tidy

# Run gen commands
../../build/operator-sdk generate k8s
../../build/operator-sdk generate crds
