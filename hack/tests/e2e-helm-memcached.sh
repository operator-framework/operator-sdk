#!/usr/bin/env bash

set -eux

source hack/lib/test_lib.sh
source hack/lib/image_lib.sh

DEST_IMAGE="quay.io/example/memcached-operator:v0.0.2"
ROOTDIR="$(pwd)"
TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT

deploy_operator() {
    kubectl create -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl create -f "$OPERATORDIR/deploy/crds/helm.example.com_memcacheds_crd.yaml"
    kubectl create -f "$OPERATORDIR/deploy/operator.yaml"
}

remove_operator() {
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/role.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/crds/helm.example.com_memcacheds_crd.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/operator.yaml"
}

test_operator() {
    # kind has an issue with certain image registries (ex. redhat's), so use a
    # different test pod image.
    local metrics_test_image="fedora:latest"

    # wait for operator pod to run
    if ! timeout 1m kubectl rollout status deployment/memcached-operator;
    then
        kubectl logs deployment/memcached-operator
        exit 1
    fi

    # verify that metrics service was created
    if ! timeout 20s bash -c -- "until kubectl get service/memcached-operator-metrics > /dev/null 2>&1; do sleep 1; done";
    then
        echo "Failed to get metrics service"
        kubectl logs deployment/memcached-operator
        exit 1
    fi

    # verify that the metrics endpoint exists
    if ! timeout 1m bash -c -- "until kubectl run --attach --rm --restart=Never test-metrics --image=$metrics_test_image -- curl -sfo /dev/null http://memcached-operator-metrics:8383/metrics; do sleep 1; done";
    then
        echo "Failed to verify that metrics endpoint exists"
        kubectl logs deployment/memcached-operator
        exit 1
    fi

    # create CR
    kubectl create -f deploy/crds/helm.example.com_v1alpha1_memcached_cr.yaml
    trap_add 'kubectl delete --ignore-not-found -f ${OPERATORDIR}/deploy/crds/helm.example.com_v1alpha1_memcached_cr.yaml' EXIT
    if ! timeout 1m bash -c -- 'until kubectl get memcachedes.helm.example.com example-memcached -o jsonpath="{..status.deployedRelease.name}" | grep "example-memcached"; do sleep 1; done';
    then
        kubectl logs deployment/memcached-operator
        exit 1
    fi

    # verify that the custom resource metrics endpoint exists
    if ! timeout 1m bash -c -- "until kubectl run --attach --rm --restart=Never test-cr-metrics --image=$metrics_test_image -- curl -sfo /dev/null http://memcached-operator-metrics:8686/metrics; do sleep 1; done";
    then
        echo "Failed to verify that custom resource metrics endpoint exists"
        kubectl logs deployment/memcached-operator
        exit 1
    fi

    release_name=$(kubectl get memcachedes.helm.example.com example-memcached -o jsonpath="{..status.deployedRelease.name}")
    memcached_deployment=$(kubectl get deployment -l "app.kubernetes.io/instance=${release_name}" -o jsonpath="{..metadata.name}")

    if ! timeout 1m kubectl rollout status deployment/${memcached_deployment};
    then
        kubectl describe pods -l "app.kubernetes.io/instance=${release_name}"
        kubectl describe deployments ${memcached_deployment}
        kubectl logs deployment/memcached-operator
        exit 1
    fi

    memcached_service=$(kubectl get service -l "app.kubernetes.io/instance=${release_name}" -o jsonpath="{..metadata.name}")
    kubectl get service ${memcached_service}

    # scale deployment replicas to 2 and verify the
    # deployment automatically scales back down to 1.
    kubectl scale deployment/${memcached_deployment} --replicas=2
    if ! timeout 1m bash -c -- "until test \$(kubectl get deployment/${memcached_deployment} -o jsonpath='{..spec.replicas}') -eq 1; do sleep 1; done";
    then
        kubectl describe pods -l "app.kubernetes.io/instance=${release_name}"
        kubectl describe deployments ${memcached_deployment}
        kubectl logs deployment/memcached-operator
        exit 1
    fi

    # update CR to replicaCount=2 and verify the deployment
    # automatically scales up to 2 replicas.
    kubectl patch memcachedes.helm.example.com example-memcached -p '[{"op":"replace","path":"/spec/replicaCount","value":2}]' --type=json
    if ! timeout 1m bash -c -- "until test \$(kubectl get deployment/${memcached_deployment} -o jsonpath='{..spec.replicas}') -eq 2; do sleep 1; done";
    then
        kubectl describe pods -l "app.kubernetes.io/instance=${release_name}"
        kubectl describe deployments ${memcached_deployment}
        kubectl logs deployment/memcached-operator
        exit 1
    fi

    kubectl delete -f deploy/crds/helm.example.com_v1alpha1_memcached_cr.yaml --wait=true
    kubectl logs deployment/memcached-operator | grep "Uninstalled release" | grep "${release_name}"
}

# create and build the operator
pushd "$TMPDIR"
log=$(operator-sdk new memcached-operator \
  --api-version=helm.example.com/v1alpha1 \
  --kind=Nginx \
  --type=helm \
  --helm-chart=stable/memcached \
  2>&1)
echo $log
if echo $log | grep -q "failed to generate RBAC rules"; then
    echo FAIL expected successful generation of RBAC rules
    exit 1
fi

pushd memcached-operator
sed -i 's|\(FROM quay.io/operator-framework/helm-operator\)\(:.*\)\?|\1:dev|g' build/Dockerfile
operator-sdk build "$DEST_IMAGE"
# If using a kind cluster, load the image into all nodes.
load_image_if_kind "$DEST_IMAGE"
sed -i "s|REPLACE_IMAGE|$DEST_IMAGE|g" deploy/operator.yaml
sed -i 's|Always|Never|g' deploy/operator.yaml
sed -i 's|hard|soft|g' helm-charts/memcached/values.yaml
sed -i 's|hard|soft|g' deploy/crds/helm.example.com_v1alpha1_memcached_cr.yaml
# kind has an issue with certain image registries (ex. redhat's), so use a
# different test pod image.
METRICS_TEST_IMAGE="fedora:latest"
docker pull "$METRICS_TEST_IMAGE"
# If using a kind cluster, load the metrics test image into all nodes.
load_image_if_kind "$METRICS_TEST_IMAGE"

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
# If using a kind cluster, load the image into all nodes.
load_image_if_kind "$DEST_IMAGE"

deploy_operator
test_operator

popd
popd
