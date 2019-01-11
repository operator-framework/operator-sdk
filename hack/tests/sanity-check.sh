#!/usr/bin/env bash
set -ex

# Make sure code syntax is correct
go vet ./...

# Make sure all returned errors are checked
./hack/ci/errcheck ./...

# Formatting checks
./hack/check_license.sh
./hack/check_error_log_msg_format.sh

# Make sure repo is in clean state
git diff --exit-code
