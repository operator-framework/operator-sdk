#!/usr/bin/env bash
set -ex

go mod tidy
go vet ./...
go fmt ./...

./hack/check_license.sh
./hack/check_error_log_msg_format.sh
./hack/doc/gen_cli_doc.sh

# Make sure repo is still in a clean state.
git diff --exit-code
