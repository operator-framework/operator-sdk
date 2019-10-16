#!/usr/bin/env bash
set -ex

# Make sure repo is in clean state before running go vet
git diff --exit-code

go vet ./...
go fmt ./...

# Ignore changes to go.mod caused by running go vet
git checkout go.mod

./hack/check_license.sh
./hack/check_error_log_msg_format.sh

# Make sure repo is still in a clean state.
git diff --exit-code
