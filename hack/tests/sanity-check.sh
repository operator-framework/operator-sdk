#!/usr/bin/env bash
set -ex

go mod tidy
go vet ./...
go fmt ./...

./hack/check-license.sh
./hack/check-error-log-msg-format.sh
make cli-doc
go run ./hack/generate/changelog/gen-changelog.go -validate-only

make install
make samples

# Make sure repo is still in a clean state.
git diff --exit-code
