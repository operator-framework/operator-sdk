#!/usr/bin/env bash
set -ex

go vet ./...
./hack/check_license.sh
./hack/check_error_case.sh

# Make sure repo is in clean state
git diff --exit-code
