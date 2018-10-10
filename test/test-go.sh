#!/usr/bin/env bash

set -e

go test ./commands/...
go test ./pkg/...
go test ./test/e2e/...
cd test/test-framework

# test framework with defaults
operator-sdk test local .

# test operator-sdk test flags
operator-sdk test local . --global-manifest deploy/crd.yaml --namespaced-manifest deploy/namespace-init.yaml --go-test-flags "-parallel 1" --kubeconfig $HOME/.kube/config

# test operator-sdk test local single namespace mode
kubectl create namespace test-memcached
operator-sdk test local . --namespace=test-memcached
kubectl delete namespace test-memcached

# go back to project root
cd ../..
go vet ./...
./hack/check_license.sh
./hack/check_error_case.sh

# Make sure repo is in clean state
git diff --exit-code
