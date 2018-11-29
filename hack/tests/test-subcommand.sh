#!/usr/bin/env bash
source hack/lib/test_lib.sh

set -ex

pushd test/test-framework
# test framework with defaults
operator-sdk test local .
# test operator-sdk test flags
operator-sdk test local . --global-manifest deploy/crds/cache_v1alpha1_memcached_crd.yaml --namespaced-manifest deploy/namespace-init.yaml --go-test-flags "-parallel 1" --kubeconfig $HOME/.kube/config --image=quay.io/coreos/operator-sdk-dev:test-framework-operator-runtime
# test operator-sdk test local single namespace mode
kubectl create namespace test-memcached
# we use the test-memcached namespace for all future tests, so we only need to set this trap once
trap_add 'kubectl delete namespace test-memcached || true' EXIT
operator-sdk test local . --namespace=test-memcached
kubectl delete namespace test-memcached
# test operator in no-setup mode
kubectl create namespace test-memcached
kubectl create -f deploy/crds/cache_v1alpha1_memcached_crd.yaml
# this runs after the popd at the end, so it needs the path from the project root
trap_add 'kubectl delete -f test/test-framework/deploy/crds/cache_v1alpha1_memcached_crd.yaml' EXIT
kubectl create -f deploy/service_account.yaml --namespace test-memcached
kubectl create -f deploy/role.yaml --namespace test-memcached
kubectl create -f deploy/role_binding.yaml --namespace test-memcached
kubectl create -f deploy/operator.yaml --namespace test-memcached
operator-sdk test local . --namespace=test-memcached --no-setup
kubectl delete namespace test-memcached
popd
