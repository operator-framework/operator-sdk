#!/usr/bin/env bash

source hack/lib/test_lib.sh

DEST_IMAGE="quay.io/example/memcached-operator:v0.0.2"

set -ex

# switch to the "default" namespace if on openshift, to match the minikube test
if which oc 2>/dev/null; then oc project default; fi

# build operator binary and base image
go build -o test/ansible-operator/ansible-operator test/ansible-operator/cmd/ansible-operator/main.go
pushd test/ansible-operator
docker build -t quay.io/water-hole/ansible-operator .
popd

# Make a test directory for Ansible tests so we avoid using default GOPATH.
# Save test directory so we can delete it on exit.
ANSIBLE_TEST_DIR="$(mktemp -d)"
trap_add 'rm -rf $ANSIBLE_TEST_DIR' EXIT
cp -a test/ansible-* "$ANSIBLE_TEST_DIR"
pushd "$ANSIBLE_TEST_DIR"

# Ansible tests should not run in a Golang environment.
unset GOPATH GOROOT

# create and build the operator
operator-sdk new memcached-operator --api-version=ansible.example.com/v1alpha1 --kind=Memcached --type=ansible
cp ansible-memcached/tasks.yml memcached-operator/roles/Memcached/tasks/main.yml
cp ansible-memcached/defaults.yml memcached-operator/roles/Memcached/defaults/main.yml
cp -a ansible-memcached/memfin memcached-operator/roles/
cat ansible-memcached/watches-finalizer.yaml >> memcached-operator/watches.yaml

pushd memcached-operator
operator-sdk build "$DEST_IMAGE"
sed -i "s|REPLACE_IMAGE|$DEST_IMAGE|g" deploy/operator.yaml
sed -i 's|Always|Never|g' deploy/operator.yaml

DIR2="$(pwd)"
# deploy the operator
kubectl create -f deploy/service_account.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/service_account.yaml' EXIT
kubectl create -f deploy/role.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/role.yaml' EXIT
kubectl create -f deploy/role_binding.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/role_binding.yaml' EXIT
kubectl create -f deploy/crds/ansible_v1alpha1_memcached_crd.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/crds/ansible_v1alpha1_memcached_crd.yaml' EXIT
kubectl create -f deploy/operator.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/operator.yaml' EXIT

# wait for operator pod to run
if ! timeout 1m kubectl rollout status deployment/memcached-operator;
then
    kubectl logs deployment/memcached-operator
    exit 1
fi

# create CR
kubectl create -f deploy/crds/ansible_v1alpha1_memcached_cr.yaml
trap_add 'kubectl delete --ignore-not-found -f ${DIR2}/deploy/crds/ansible_v1alpha1_memcached_cr.yaml' EXIT
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

# make a configmap that the finalizer should remove
kubectl create configmap deleteme
trap_add 'kubectl delete --ignore-not-found configmap deleteme' EXIT

kubectl delete -f ${DIR2}/deploy/crds/ansible_v1alpha1_memcached_cr.yaml --wait=true
# if the finalizer did not delete the configmap...
if kubectl get configmap deleteme;
then
    echo FAIL: the finalizer did not delete the configmap
    kubectl logs deployment/memcached-operator
    exit 1
fi

# The deployment should get garbage collected, so we expect to fail getting the deployment.
if ! timeout 20s bash -c -- "while kubectl get deployment ${memcached_deployment}; do sleep 1; done";
then
    kubectl logs deployment/memcached-operator
    exit 1
fi


## TODO enable when this is fixed: https://github.com/operator-framework/operator-sdk/issues/818
# if kubectl logs deployment/memcached-operator | grep -i error;
# then
    # echo FAIL: the operator log includes errors
    # kubectl logs deployment/memcached-operator
    # exit 1
# fi

popd
popd
