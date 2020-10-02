#!/usr/bin/env bash
set -ex

go mod tidy
go vet ./...
go fmt ./...

./hack/check-license.sh
./hack/check-error-log-msg-format.sh
./hack/generate/cli-doc/gen-cli-doc.sh
go run ./hack/generate/changelog/gen-changelog.go -validate-only

make install
go run ./hack/generate/samples/generate_all.go

# Make sure repo is still in a clean state.
# Note that we are ignoring helm manifests with roles.
# More info: https://github.com/operator-framework/operator-sdk/issues/3873
git diff --exit-code -- . ':!testdata/helm/memcached-operator/bundle/manifests/memcached-operator.clusterserviceversion.yaml' ':!testdata/helm/memcached-operator/config/rbac/role.yaml'
