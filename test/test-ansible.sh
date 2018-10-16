#!/usr/bin/env bash

DEST_IMAGE="quay.io/example/memcached-operator:v0.0.2"

set -evx

# switch to the "default" namespace if on openshift, to match the minikube test
if which oc 2>/dev/null; then oc project default; fi

# build operator binary and base image
go build -o test/ansible-operator/ansible-operator test/ansible-operator/cmd/ansible-operator/main.go
pushd test
pushd ansible-operator
docker build -t quay.io/water-hole/ansible-operator .
popd

# create and build the operator
operator-sdk new memcached-operator --api-version=ansible.example.com/v1alpha1 --kind=Memcached --type=ansible
cp ansible-memcached/tasks.yml memcached-operator/roles/Memcached/tasks/main.yml
cp ansible-memcached/defaults.yml memcached-operator/roles/Memcached/defaults/main.yml
cp -a ansible-memcached/memfin memcached-operator/roles/
cat ansible-memcached/watches-finalizer.yaml >> memcached-operator/watches.yaml

pushd memcached-operator
operator-sdk build $DEST_IMAGE
sed -i "s|REPLACE_IMAGE|$DEST_IMAGE|g" deploy/operator.yaml
sed -i 's|Always|Never|g' deploy/operator.yaml

# deploy the operator
kubectl create -f deploy/service_account.yaml
kubectl create -f deploy/role.yaml
kubectl create -f deploy/role_binding.yaml
kubectl create -f deploy/crds/ansible_v1alpha1_memcached_crd.yaml
kubectl create -f deploy/operator.yaml

# wait for operator pod to run
if ! timeout 1m kubectl rollout status deployment/memcached-operator;
then
    kubectl logs deployment/memcached-operator
    exit 1
fi

# create CR
kubectl create -f deploy/crds/ansible_v1alpha1_memcached_cr.yaml
if ! timeout 20s bash -c -- 'until kubectl get deployment -l app=memcached | grep memcached; do sleep 1; done';
then
    kubectl logs deployment/memcached-operator
    exit 1
fi
memcached_deployment=$(kubectl get deployment -l app=memcached -o jsonpath="{..metadata.name}")
if ! timeout 1m kubectl rollout status deployment/${memcached_deployment};
then
    kubectl logs deployment/${memcached_deployment}
    exit 1
fi

# Test finalizer
kubectl delete -f deploy/crds/ansible_v1alpha1_memcached_cr.yaml --wait=true
kubectl logs deployment/memcached-operator | grep "this is a finalizer"

popd
popd
