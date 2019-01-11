#!/usr/bin/env bash
set -ex

go vet ./...
./hack/check_license.sh
./hack/check_error_log_msg_format.sh

# Make sure repo is in clean state
git diff --exit-code
