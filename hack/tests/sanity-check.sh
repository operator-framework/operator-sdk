#!/usr/bin/env bash
set -ex

go mod tidy
go vet ./...
go fmt ./...

./hack/check-license.sh
./hack/check-error-log-msg-format.sh
./hack/generate/gen-cli-doc.sh
./hack/generate/gen-test-framework.sh

# Make sure repo is still in a clean state.
git diff --exit-code
