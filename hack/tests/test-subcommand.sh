#!/usr/bin/env bash
set -ex

cd test/test-framework
# test framework with defaults
operator-sdk test local ./test/e2e
# test operator-sdk test flags
operator-sdk test local ./test/e2e --global-manifest deploy/crds/cache_v1alpha1_memcached_crd.yaml --namespaced-manifest deploy/namespace-init.yaml --go-test-flags "-parallel 1" --kubeconfig $HOME/.kube/config
# test operator-sdk test local single namespace mode
kubectl create namespace test-memcached
operator-sdk test local ./test/e2e --namespace=test-memcached
kubectl delete namespace test-memcached
# test operator in up local mode
kubectl create namespace test-memcached
operator-sdk test local ./test/e2e --up-local --namespace=test-memcached
kubectl delete namespace test-memcached
