#!/usr/bin/env bash
set -ex

go mod tidy
go vet ./...
go fmt ./...

./hack/check-license.sh
./hack/check-error-log-msg-format.sh
./hack/generate/cli-doc/gen-cli-doc.sh
./hack/generate/test-framework/gen-test-framework.sh
go run ./hack/generate/changelog/gen-changelog.go -validate-only

# Make sure repo is still in a clean state.
git diff --exit-code
