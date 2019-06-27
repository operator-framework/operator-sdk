#!/usr/bin/env bash

source hack/lib/test_lib.sh

set -eux

# ansible proxy test require a running cluster; run during e2e instead
go test -count=1 ./pkg/ansible/proxy/...

DEST_IMAGE="quay.io/example/memcached-operator:v0.0.2"
ROOTDIR="$(pwd)"
GOTMP="$(mktemp -d)"
trap_add 'rm -rf $GOTMP' EXIT

deploy_operator() {
    kubectl create -f "$OPERATORDIR/deploy/service_account.yaml"
    oc adm policy add-cluster-role-to-user cluster-admin -z memcached-operator || :
    kubectl create -f "$OPERATORDIR/deploy/role.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl create -f "$OPERATORDIR/deploy/crds/ansible_v1alpha1_memcached_crd.yaml"
    kubectl create -f "$OPERATORDIR/deploy/crds/ansible_v1alpha1_foo_crd.yaml"
    kubectl create -f "$OPERATORDIR/deploy/operator.yaml"
}

remove_operator() {
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/role.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/crds/ansible_v1alpha1_memcached_crd.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/crds/ansible_v1alpha1_foo_crd.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/operator.yaml"
}

test_operator() {
    # wait for operator pod to run
    if ! timeout 1m kubectl rollout status deployment/memcached-operator;
    then
        echo FAIL: operator failed to run
        kubectl logs deployment/memcached-operator -c operator
        kubectl logs deployment/memcached-operator -c ansible
        exit 1
    fi

    # create CR
    kubectl create -f deploy/crds/ansible_v1alpha1_memcached_cr.yaml
    if ! timeout 20s bash -c -- 'until kubectl get deployment -l app=memcached | grep memcached; do sleep 1; done';
    then
        echo FAIL: operator failed to create memcached Deployment
        kubectl logs deployment/memcached-operator -c operator
        kubectl logs deployment/memcached-operator -c ansible
        exit 1
    fi
    memcached_deployment=$(kubectl get deployment -l app=memcached -o jsonpath="{..metadata.name}")
    if ! timeout 1m kubectl rollout status deployment/${memcached_deployment};
    then
        echo FAIL: memcached Deployment failed rollout
        kubectl logs deployment/${memcached_deployment}
        exit 1
    fi


    # make a configmap that the finalizer should remove
    kubectl create configmap deleteme
    trap_add 'kubectl delete --ignore-not-found configmap deleteme' EXIT

    kubectl delete -f ${OPERATORDIR}/deploy/crds/ansible_v1alpha1_memcached_cr.yaml --wait=true
    # if the finalizer did not delete the configmap...
    if kubectl get configmap deleteme 2> /dev/null;
    then
        echo FAIL: the finalizer did not delete the configmap
        kubectl logs deployment/memcached-operator -c operator
        kubectl logs deployment/memcached-operator -c ansible
        exit 1
    fi

    # The deployment should get garbage collected, so we expect to fail getting the deployment.
    if ! timeout 20s bash -c -- "while kubectl get deployment ${memcached_deployment} 2> /dev/null; do sleep 1; done";
    then
        echo FAIL: memcached Deployment did not get garbage collected
        kubectl logs deployment/memcached-operator -c operator
        kubectl logs deployment/memcached-operator -c ansible
        exit 1
    fi

    # Ensure that no errors appear in the log
    if kubectl logs deployment/memcached-operator -c operator | grep -i error;
    then
        echo FAIL: the operator log includes errors
        kubectl logs deployment/memcached-operator -c operator
        kubectl logs deployment/memcached-operator -c ansible
        exit 1
    fi
}

# switch to the "default" namespace if on openshift, to match the minikube test
if which oc 2>/dev/null; then oc project default; fi

# create and build the operator
pushd "$GOTMP"
operator-sdk new memcached-operator \
  --api-version=ansible.example.com/v1alpha1 \
  --kind=Memcached \
  --type=ansible
cp "$ROOTDIR/test/ansible-memcached/tasks.yml" memcached-operator/roles/memcached/tasks/main.yml
cp "$ROOTDIR/test/ansible-memcached/defaults.yml" memcached-operator/roles/memcached/defaults/main.yml
cp -a "$ROOTDIR/test/ansible-memcached/memfin" memcached-operator/roles/
cat "$ROOTDIR/test/ansible-memcached/watches-finalizer.yaml" >> memcached-operator/watches.yaml
# Append Foo kind to watches to test watching multiple Kinds
cat "$ROOTDIR/test/ansible-memcached/watches-foo-kind.yaml" >> memcached-operator/watches.yaml

pushd memcached-operator
# Add a second Kind to test watching multiple GVKs
operator-sdk add crd --kind=Foo --api-version=ansible.example.com/v1alpha1
sed -i 's|\(FROM quay.io/operator-framework/ansible-operator\)\(:.*\)\?|\1:dev|g' build/Dockerfile
operator-sdk build "$DEST_IMAGE"
sed -i "s|{{ REPLACE_IMAGE }}|$DEST_IMAGE|g" deploy/operator.yaml
sed -i 's|{{ pull_policy.default..Always.. }}|Never|g' deploy/operator.yaml

OPERATORDIR="$(pwd)"

deploy_operator
trap_add 'remove_operator' EXIT
test_operator
remove_operator

echo "###"
echo "### Base image testing passed"
echo "### Now testing migrate to hybrid operator"
echo "###"

export GO111MODULE=on
operator-sdk migrate --repo=github.com/example-inc/memcached-operator

if [[ ! -e build/Dockerfile.sdkold ]];
then
    echo FAIL the old Dockerfile should have been renamed to Dockerfile.sdkold
    exit 1
fi

add_go_mod_replace "github.com/operator-framework/operator-sdk" "$ROOTDIR"
# Build the project to resolve dependency versions in the modfile.
go build ./...

operator-sdk build "$DEST_IMAGE"

deploy_operator
test_operator

popd
popd
