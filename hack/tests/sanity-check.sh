#!/usr/bin/env bash
set -ex

go mod tidy
go vet ./...
go fmt ./...

./hack/check_license.sh
./hack/check_error_log_msg_format.sh
./hack/doc/gen_cli_doc.sh

# Make sure that test/test-framework is updated according to the current source code
make install
cd test/test-framework

# Create files in the mock test/test-framework just for we are able to exec the commands
cat > build/Dockerfile
cat > go.mod

GO111MODULE=on operator-sdk generate k8s
GO111MODULE=on operator-sdk generate openapi

# Remove files that are required and created just for we exec the commands.
rm -rf build/Dockerfile
rm -rf go.mod
rm -rf go.sum

# Make sure repo is still in a clean state.
git diff --exit-code
