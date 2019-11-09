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

# Create the required files in the mock test/test-framework just to allow run from it the oprator-sdk commands
echo > $DOCKERFILE
go mod init test-framework

# Remove files that are required and created just to exec the commands.
trap_add 'rm -rf $DOCKERFILE go.mod go.sum' EXIT

# Run gen commands
operator-sdk generate k8s
operator-sdk generate openapi
