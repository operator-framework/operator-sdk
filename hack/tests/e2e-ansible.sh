#!/usr/bin/env bash

set -eu

source hack/lib/test_lib.sh
source hack/lib/image_lib.sh
source ./hack/lib/common.sh

# ansible proxy test require a running cluster; run during e2e instead
go test -count=1 ./pkg/ansible/proxy/...

DEST_IMAGE="quay.io/example/memcached-operator:v0.0.2"
ROOTDIR="$(pwd)"
TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT

deploy_operator() {
    header_text "Running deploy operator"
    kubectl create -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl create -f "$OPERATORDIR/deploy/crds/ansible.example.com_memcacheds_crd.yaml"
    kubectl create -f "$OPERATORDIR/deploy/crds/ansible.example.com_foos_crd.yaml"
    kubectl create -f "$OPERATORDIR/deploy/operator.yaml"
}

remove_operator() {
    header_text "Running remove operator"
    kubectl delete --wait=true --ignore-not-found=true --timeout=2m -f "$OPERATORDIR/deploy/crds/ansible.example.com_memcacheds_crd.yaml"
    kubectl delete --wait=true --ignore-not-found=true --timeout=2m -f "$OPERATORDIR/deploy/crds/ansible.example.com_foos_crd.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/operator.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/role.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/role_binding.yaml"
}

operator_logs() {
    header_text "Getting Pod logs"
    kubectl describe pods
    header_text "Getting events"
    kubectl get events
    header_text "Getting operator logs"
    kubectl logs deployment/memcached-operator -c operator
    header_text "Getting Ansible logs"
    kubectl logs deployment/memcached-operator -c ansible
}

test_operator() {
    header_text "Testing operator metrics"
    # kind has an issue with certain image registries (ex. redhat's), so use a
    # different test pod image.
    local metrics_test_image="fedora:latest"

    header_text "wait for operator pod to run"
    if ! timeout 1m kubectl rollout status deployment/memcached-operator;
    then
        error_text "FAIL: Failed to run"
        operator_logs
        exit 1
    fi

    header_text "verify that metrics service was created"
    if ! timeout 60s bash -c -- "until kubectl get service/memcached-operator-metrics > /dev/null 2>&1; do sleep 1; done";
    then
        error_text "FAIL: Failed to get metrics service"
        operator_logs
        exit 1
    fi

    header_text "verify that the metrics endpoint exists (Port 8383)"
    if ! timeout 1m bash -c -- "until kubectl run --attach --rm --restart=Never test-metrics --image=$metrics_test_image -- curl -sfo /dev/null http://memcached-operator-metrics:8383/metrics; do sleep 1; done";
    then
        error_text "FAIL: Failed to verify that metrics endpoint exists"
        operator_logs
        exit 1
    fi

    header_text "verify that the metrics endpoint exists (Port 8686)"
    if ! timeout 1m bash -c -- "until kubectl run --attach --rm --restart=Never test-metrics --image=$metrics_test_image -- curl -sfo /dev/null http://memcached-operator-metrics:8686/metrics; do sleep 1; done";
    then
        error_text "FAIL: Failed to verify that metrics endpoint exists"
        operator_logs
        exit 1
    fi

    header_text "create custom resource (Memcached CR)"
    kubectl create -f deploy/crds/ansible.example.com_v1alpha1_memcached_cr.yaml
    if ! timeout 60s bash -c -- 'until kubectl get deployment -l app=memcached | grep memcached; do sleep 1; done';
    then
        error_text "FAIL: Failed to verify to create memcached Deployment"
        operator_logs
        exit 1
    fi

    header_text "verify that metrics reflect cr creation"
    if ! timeout 1m bash -c -- "until kubectl run -it --rm --restart=Never test-metrics --image=$metrics_test_image -- curl http://memcached-operator-metrics:8686/metrics | grep example-memcached; do sleep 1; done";
    then
        error_text "FAIL: Failed to verify custom resource metrics"
        operator_logs
        exit 1
    fi

    header_text "get memcached deploy by labels"
    memcached_deployment=$(kubectl get deployment -l app=memcached -o jsonpath="{..metadata.name}")
    if ! timeout 1m kubectl rollout status deployment/${memcached_deployment};
    then
        error_text "FAIL: Failed memcached Deployment failed rollout"
        kubectl logs deployment/${memcached_deployment}
        exit 1
    fi

    header_text "create a configmap that the finalizer should remove"
    kubectl create configmap deleteme
    trap_add 'kubectl delete --ignore-not-found configmap deleteme' EXIT

    header_text "delete custom resource (Memcached CR)"
    kubectl delete -f ${OPERATORDIR}/deploy/crds/ansible.example.com_v1alpha1_memcached_cr.yaml --wait=true
    header_text "if the finalizer did not delete the configmap..."
    if kubectl get configmap deleteme 2> /dev/null;
    then
        error_text "FAIL: the finalizer did not delete the configmap"
        operator_logs
        exit 1
    fi

    header_text "The deployment should get garbage collected, so we expect to fail getting the deployment."
    if ! timeout 60s bash -c -- "while kubectl get deployment ${memcached_deployment} 2> /dev/null; do sleep 1; done";
    then
        error_text "FAIL: memcached Deployment did not get garbage collected"
        operator_logs
        exit 1
    fi

    header_text "Ensure that no errors appear in the log"
    if kubectl logs deployment/memcached-operator -c operator | grep -i error;
    then
        error_text "FAIL: the operator log includes errors"
        operator_logs
        exit 1
    fi
}

header_text "Creating and building the operator"
pushd "$TMPDIR"
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

header_text "Adding a second Kind to test watching multiple GVKs"
operator-sdk add crd --kind=Foo --api-version=ansible.example.com/v1alpha1
sed -i".bak" -E -e 's/(FROM quay.io\/operator-framework\/ansible-operator)(:.*)?/\1:dev/g' build/Dockerfile; rm -f build/Dockerfile.bak
operator-sdk build "$DEST_IMAGE"
# If using a kind cluster, load the image into all nodes.
load_image_if_kind "$DEST_IMAGE"
sed -i".bak" -E -e "s|\{\{ REPLACE_IMAGE \}\}|$DEST_IMAGE|g" deploy/operator.yaml; rm -f deploy/operator.yaml.bak
sed -i".bak" -E -e 's|\{\{ pull_policy.default..Always.. \}\}|Never|g' deploy/operator.yaml; rm -f deploy/operator.yaml.bak
# kind has an issue with certain image registries (ex. redhat's), so use a
# different test pod image.
METRICS_TEST_IMAGE="fedora:latest"
docker pull "$METRICS_TEST_IMAGE"
# If using a kind cluster, load the metrics test image into all nodes.
load_image_if_kind "$METRICS_TEST_IMAGE"

OPERATORDIR="$(pwd)"

trap_add 'remove_operator' EXIT
deploy_operator
test_operator
remove_operator

header_text "###"
header_text "### Base image testing passed"
header_text "### Now testing migrate to hybrid operator"
header_text "###"

operator-sdk migrate --repo=github.com/example-inc/memcached-operator

if [[ ! -e build/Dockerfile.sdkold ]];
then
    error_text "FAIL the old Dockerfile should have been renamed to Dockerfile.sdkold"
    exit 1
fi

add_go_mod_replace "github.com/operator-framework/operator-sdk" "$ROOTDIR"
header_text "Build the project to resolve dependency versions in the modfile."
go build ./...

operator-sdk build "$DEST_IMAGE"

header_text "If using a kind cluster, load the image into all nodes."
load_image_if_kind "$DEST_IMAGE"

deploy_operator
test_operator

popd
popd
