#!/usr/bin/env bash

source hack/lib/test_lib.sh

set -eux

DEST_IMAGE="quay.io/example/nginx-operator:v0.0.2"
ROOTDIR="$(pwd)"
GOTMP="$(mktemp -d)"
trap_add 'rm -rf $GOTMP' EXIT

deploy_operator() {
    kubectl create -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl create -f "$OPERATORDIR/deploy/crds/helm_v1alpha1_nginx_crd.yaml"
    kubectl create -f "$OPERATORDIR/deploy/operator.yaml"
}

remove_operator() {
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/role.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/crds/helm_v1alpha1_nginx_crd.yaml"
    kubectl delete --ignore-not-found=true -f "$OPERATORDIR/deploy/operator.yaml"
}

test_operator() {
    # wait for operator pod to run
    if ! timeout 1m kubectl rollout status deployment/nginx-operator;
    then
        kubectl logs deployment/nginx-operator
        exit 1
    fi

    # verify that metrics service was created
    if ! timeout 20s bash -c -- "until kubectl get service/nginx-operator-metrics > /dev/null 2>&1; do sleep 1; done";
    then
        echo "Failed to get metrics service"
        kubectl logs deployment/nginx-operator
        exit 1
    fi

    # verify that the metrics endpoint exists
    if ! timeout 1m bash -c -- "until kubectl run -it --rm --restart=Never test-metrics --image=registry.access.redhat.com/ubi7/ubi-minimal:latest -- curl -sfo /dev/null http://nginx-operator-metrics:8383/metrics; do sleep 1; done";
    then
        echo "Failed to verify that metrics endpoint exists"
        kubectl logs deployment/nginx-operator
        exit 1
    fi

    # create CR
    kubectl create -f deploy/crds/helm_v1alpha1_nginx_cr.yaml
    trap_add 'kubectl delete --ignore-not-found -f ${OPERATORDIR}/deploy/crds/helm_v1alpha1_nginx_cr.yaml' EXIT
    if ! timeout 1m bash -c -- 'until kubectl get nginxes.helm.example.com example-nginx -o jsonpath="{..status.deployedRelease.name}" | grep "example-nginx"; do sleep 1; done';
    then
        kubectl logs deployment/nginx-operator
        exit 1
    fi

    # verify that the custom resource metrics endpoint exists
    if ! timeout 1m bash -c -- "until kubectl run -it --rm --restart=Never test-cr-metrics --image=registry.access.redhat.com/ubi7/ubi-minimal:latest -- curl -sfo /dev/null http://nginx-operator-metrics:8686/metrics; do sleep 1; done";
    then
        echo "Failed to verify that custom resource metrics endpoint exists"
        kubectl logs deployment/nginx-operator
        exit 1
    fi

    release_name=$(kubectl get nginxes.helm.example.com example-nginx -o jsonpath="{..status.deployedRelease.name}")
    nginx_deployment=$(kubectl get deployment -l "app.kubernetes.io/instance=${release_name}" -o jsonpath="{..metadata.name}")

    if ! timeout 1m kubectl rollout status deployment/${nginx_deployment};
    then
        kubectl describe pods -l "app.kubernetes.io/instance=${release_name}"
        kubectl describe deployments ${nginx_deployment}
        kubectl logs deployment/nginx-operator
        exit 1
    fi

    nginx_service=$(kubectl get service -l "app.kubernetes.io/instance=${release_name}" -o jsonpath="{..metadata.name}")
    kubectl get service ${nginx_service}

    # scale deployment replicas to 2 and verify the
    # deployment automatically scales back down to 1.
    kubectl scale deployment/${nginx_deployment} --replicas=2
    if ! timeout 1m bash -c -- "until test \$(kubectl get deployment/${nginx_deployment} -o jsonpath='{..spec.replicas}') -eq 1; do sleep 1; done";
    then
        kubectl describe pods -l "app.kubernetes.io/instance=${release_name}"
        kubectl describe deployments ${nginx_deployment}
        kubectl logs deployment/nginx-operator
        exit 1
    fi

    # update CR to replicaCount=2 and verify the deployment
    # automatically scales up to 2 replicas.
    kubectl patch nginxes.helm.example.com example-nginx -p '[{"op":"replace","path":"/spec/replicaCount","value":2}]' --type=json
    if ! timeout 1m bash -c -- "until test \$(kubectl get deployment/${nginx_deployment} -o jsonpath='{..spec.replicas}') -eq 2; do sleep 1; done";
    then
        kubectl describe pods -l "app.kubernetes.io/instance=${release_name}"
        kubectl describe deployments ${nginx_deployment}
        kubectl logs deployment/nginx-operator
        exit 1
    fi

    kubectl delete -f deploy/crds/helm_v1alpha1_nginx_cr.yaml --wait=true
    kubectl logs deployment/nginx-operator | grep "Uninstalled release" | grep "${release_name}"
}

# if on openshift switch to the "default" namespace
# and allow containers to run as root (necessary for
# default nginx image)
if which oc 2>/dev/null;
then
    oc project default
    oc adm policy add-scc-to-user anyuid -z default
fi


# create and build the operator
pushd "$GOTMP"
log=$(operator-sdk new nginx-operator \
  --api-version=helm.example.com/v1alpha1 \
  --kind=Nginx \
  --type=helm \
  2>&1)
echo $log
if echo $log | grep -q "failed to generate RBAC rules"; then
    echo FAIL expected successful generation of RBAC rules
    exit 1
fi

pushd nginx-operator
sed -i 's|\(FROM quay.io/operator-framework/helm-operator\)\(:.*\)\?|\1:dev|g' build/Dockerfile
operator-sdk build "$DEST_IMAGE"
sed -i "s|REPLACE_IMAGE|$DEST_IMAGE|g" deploy/operator.yaml
sed -i 's|Always|Never|g' deploy/operator.yaml

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
operator-sdk migrate --repo=github.com/example-inc/nginx-operator

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
