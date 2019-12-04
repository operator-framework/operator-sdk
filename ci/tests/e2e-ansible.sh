#!/usr/bin/env bash

source hack/lib/test_lib.sh

set -eux

component="osdk-ansible-e2e"
eval IMAGE=$IMAGE_FORMAT
component="osdk-ansible-e2e-hybrid"
eval IMAGE2=$IMAGE_FORMAT
ROOTDIR="$(pwd)"
TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT

mkdir -p $ROOTDIR/bin
export PATH=$ROOTDIR/bin:$PATH

if ! [ -x "$(command -v kubectl)" ]; then
    curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.15.4/bin/linux/amd64/kubectl && chmod +x kubectl && mv kubectl bin/
fi

if ! [ -x "$(command -v oc)" ]; then
    curl -Lo oc.tar.gz https://github.com/openshift/origin/releases/download/v3.11.0/openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit.tar.gz
    tar xvzOf oc.tar.gz openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit/oc > oc && chmod +x oc && mv oc bin/ && rm oc.tar.gz
fi

oc version

make install

deploy_operator() {
    kubectl create -f "$OPERATORDIR/deploy/service_account.yaml"
    if oc api-versions | grep openshift; then
        oc adm policy add-cluster-role-to-user cluster-admin -z memcached-operator || :
    fi
    kubectl create -f "$OPERATORDIR/deploy/role.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl create -f "$OPERATORDIR/deploy/crds/ansible.example.com_memcacheds_crd.yaml"
    kubectl create -f "$OPERATORDIR/deploy/crds/ansible.example.com_foos_crd.yaml"
    kubectl create -f "$OPERATORDIR/deploy/operator.yaml"
}

remove_operator() {
    kubectl delete --wait=true --ignore-not-found=true --timeout=60s -f "$OPERATORDIR/deploy/crds/ansible.example.com_memcacheds_crd.yaml"
    kubectl delete --wait=true --ignore-not-found=true --timeout=60s -f "$OPERATORDIR/deploy/crds/ansible.example.com_foos_crd.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/operator.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/role.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/role_binding.yaml"
}

operator_logs() {
    kubectl describe pods
    kubectl logs deployment/memcached-operator -c operator
    kubectl logs deployment/memcached-operator -c ansible
}


test_operator() {
    local metrics_test_image="registry.access.redhat.com/ubi8/ubi-minimal:latest"

    # wait for operator pod to run
    if ! timeout 1m kubectl rollout status deployment/memcached-operator;
    then
        echo FAIL: operator failed to run
        operator_logs
        exit 1
    fi

    # verify that metrics service was created
    if ! timeout 60s bash -c -- "until kubectl get service/memcached-operator-metrics > /dev/null 2>&1; do sleep 1; done";
    then
        echo "Failed to get metrics service"
        operator_logs
        exit 1
    fi

    # verify that the metrics endpoint exists
    if ! timeout 1m bash -c -- "until kubectl run --attach --rm --restart=Never test-metrics --image=$metrics_test_image -- curl -sfo /dev/null http://memcached-operator-metrics:8383/metrics; do sleep 1; done";
    then
        echo "Failed to verify that metrics endpoint exists"
        operator_logs
        exit 1
    fi

    # verify that the operator metrics endpoint exists
    if ! timeout 1m bash -c -- "until kubectl run --attach --rm --restart=Never test-metrics --image=$metrics_test_image -- curl -sfo /dev/null http://memcached-operator-metrics:8686/metrics; do sleep 1; done";
    then
        echo "Failed to verify that metrics endpoint exists"
        operator_logs
        exit 1
    fi

    # create CR
    kubectl create -f deploy/crds/ansible.example.com_v1alpha1_memcached_cr.yaml
    if ! timeout 60s bash -c -- 'until kubectl get deployment -l app=memcached | grep memcached; do sleep 1; done';
    then
        echo FAIL: operator failed to create memcached Deployment
        operator_logs
        exit 1
    fi

    # verify that metrics reflect cr creation
    if ! timeout 1m bash -c -- "until kubectl run -it --rm --restart=Never test-metrics --image=$metrics_test_image -- curl http://memcached-operator-metrics:8686/metrics | grep example-memcached; do sleep 1; done";
    then
        echo "Failed to verify custom resource metrics"
        operator_logs
        exit 1
    fi

    memcached_deployment=$(kubectl get deployment -l app=memcached -o jsonpath="{..metadata.name}")
    if ! timeout 1m kubectl rollout status deployment/${memcached_deployment};
    then
        echo FAIL: memcached Deployment failed rollout
        operator_logs
        kubectl logs deployment/${memcached_deployment}
        exit 1
    fi


    # make a configmap that the finalizer should remove
    kubectl create configmap deleteme
    trap_add 'kubectl delete --ignore-not-found configmap deleteme' EXIT

    kubectl delete -f ${OPERATORDIR}/deploy/crds/ansible.example.com_v1alpha1_memcached_cr.yaml --wait=true
    # if the finalizer did not delete the configmap...
    if kubectl get configmap deleteme 2> /dev/null;
    then
        echo FAIL: the finalizer did not delete the configmap
        operator_logs
        exit 1
    fi

    # The deployment should get garbage collected, so we expect to fail getting the deployment.
    if ! timeout 60s bash -c -- "while kubectl get deployment ${memcached_deployment} 2> /dev/null; do sleep 1; done";
    then
        echo FAIL: memcached Deployment did not get garbage collected
        operator_logs
        exit 1
    fi

    # Ensure that no errors appear in the log
    if kubectl logs deployment/memcached-operator -c operator | grep -i error;
    then
        echo FAIL: the operator log includes errors
        operator_logs
        exit 1
    fi
}

# switch to the "default" namespace
oc project default

# create and build the operator
pushd "$TMPDIR"
operator-sdk new memcached-operator --api-version=ansible.example.com/v1alpha1 --kind=Memcached --type=ansible

pushd memcached-operator
# Add a second Kind to test watching multiple GVKs
operator-sdk add crd --kind=Foo --api-version=ansible.example.com/v1alpha1
sed -i 's|{{ pull_policy.default..Always.. }}|Always|g' deploy/operator.yaml
cp deploy/operator.yaml deploy/operator-copy.yaml
sed -i "s|{{ REPLACE_IMAGE }}|$IMAGE|g" deploy/operator.yaml

OPERATORDIR="$(pwd)"

deploy_operator
trap_add 'remove_operator' EXIT
test_operator
remove_operator

# the memcached-operator pods remain after the deployment is gone; wait until the pods are removed
if ! timeout 60s bash -c -- "until kubectl get pods -l name=memcached-operator |& grep \"No resources found\"; do sleep 2; done";
then
    echo FAIL: memcached-operator Deployment did not get garbage collected
    operator_logs
    exit 1
fi

cp deploy/operator-copy.yaml deploy/operator.yaml
sed -i "s|{{ REPLACE_IMAGE }}|$IMAGE2|g" deploy/operator.yaml
deploy_operator
test_operator
remove_operator

popd
