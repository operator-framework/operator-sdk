#!/usr/bin/env bash
set -ex

# Make sure repo is in clean state before running go tools
git diff --exit-code

go mod tidy
go vet ./...
go fmt ./...

./hack/check_license.sh
./hack/check_error_log_msg_format.sh

# Ignore changes to go.mod and go.sum
git checkout go.mod go.sum

# Make sure repo is still in a clean state.
git diff --exit-code
