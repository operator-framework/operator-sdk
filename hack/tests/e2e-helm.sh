#!/usr/bin/env bash

set -eux

source hack/lib/test_lib.sh
source hack/lib/image_lib.sh

DEST_IMAGE="quay.io/example/nginx-operator:v0.0.2"
TMPDIR="$(mktemp -d)"
pushd $TMPDIR
trap_add 'rm -rf $TMPDIR' EXIT

setup_envs $tmp_sdk_root

# kind has an issue with certain image registries (ex. redhat's), so use a
# different test pod image.
METRICS_TEST_IMAGE="curlimages/curl:latest"

test_namespace="nginx-cr-system"
operator_namespace="nginx-operator-system"

deploy_operator() {
    make install
    make deploy IMG="$DEST_IMAGE"
    kubectl create clusterrolebinding nginx-operator-system-metrics-reader --clusterrole=nginx-operator-metrics-reader --serviceaccount=${operator_namespace}:default
    kubectl create namespace ${test_namespace}
}

remove_operator() {
    kubectl delete --ignore-not-found=true --namespace=${test_namespace} -f "$OPERATORDIR/config/samples/helm.example_v1alpha1_nginx.yaml"
    kubectl delete --ignore-not-found=true namespace ${test_namespace}
    kubectl delete --ignore-not-found=true clusterrolebinding nginx-operator-system-metrics-reader
    make undeploy
}

operator_logs() {
    kubectl get all --namespace=${operator_namespace}
    kubectl get events --namespace=${operator_namespace}
    kubectl logs deployment/nginx-operator-controller-manager -c manager --namespace=${operator_namespace}
}

test_operator() {
    # kind has an issue with certain image registries (ex. redhat's), so use a
    # different test pod image.
    local metrics_test_image="$METRICS_TEST_IMAGE"

    # wait for operator pod to run
    if ! timeout 1m kubectl rollout status deployment/nginx-operator-controller-manager -n ${operator_namespace} ;
    then
        error_text "Failed to rollout status"
        operator_logs
        exit 1
    fi

    metrics_service="nginx-operator-controller-manager-metrics-service"

    # verify that metrics service was created
    if ! timeout 60s bash -c -- "until kubectl get service/${metrics_service} --namespace=${operator_namespace} > /dev/null 2>&1; do sleep 1; done";
    then
        error_text "Failed to get metrics service"
        operator_logs
        exit 1
    fi


    serviceaccount_secret=$(kubectl get serviceaccounts default -n ${operator_namespace} -o jsonpath='{.secrets[0].name}')
    token=$(kubectl get secret ${serviceaccount_secret} -n ${operator_namespace} -o jsonpath={.data.token} | base64 -d)

    # verify that the metrics endpoint exists
    if ! timeout 60s bash -c -- "until kubectl run --attach --rm --restart=Never --namespace=${operator_namespace} test-metrics --image=${metrics_test_image} -- -sfkH \"Authorization: Bearer ${token}\" https://${metrics_service}:8443/metrics; do sleep 1; done";
    then
        error_text "Failed to verify that metrics endpoint exists"
        operator_logs
        exit 1
    fi

    # create CR
    kubectl create --namespace=${test_namespace} -f config/samples/helm.example_v1alpha1_nginx.yaml
    trap_add "kubectl delete --namespace=${test_namespace} --ignore-not-found -f ${OPERATORDIR}/config/samples/helm.example_v1alpha1_nginx.yaml" EXIT
    if ! timeout 1m bash -c -- "until kubectl get --namespace=${test_namespace} nginxes.helm.example.com nginx-sample -o jsonpath='{..status.deployedRelease.name}' | grep 'nginx-sample'; do sleep 1; done";
    then
        error_text "Failed to create CR"
        operator_logs
        exit 1
    fi



    release_name=$(kubectl get --namespace=${test_namespace} nginxes.helm.example.com nginx-sample -o jsonpath="{..status.deployedRelease.name}")
    nginx_deployment=$(kubectl get --namespace=${test_namespace}  deployment -l "app.kubernetes.io/instance=${release_name}" -o jsonpath="{..metadata.name}")

    if ! timeout 1m kubectl rollout --namespace=${test_namespace} status deployment/${nginx_deployment};
    then
        error_text "FAIL: kubectl rollout status CR deployment"
        kubectl describe --namespace=${test_namespace} pods -l "app.kubernetes.io/instance=${release_name}"
        kubectl describe --namespace=${test_namespace} deployments ${nginx_deployment}
        operator_logs
        exit 1
    fi

    nginx_service=$(kubectl get --namespace=${test_namespace} service -l "app.kubernetes.io/instance=${release_name}" -o jsonpath="{..metadata.name}")
    kubectl get --namespace=${test_namespace} service ${nginx_service}

    # scale deployment replicas to 2 and verify the
    # deployment automatically scales back down to 1.
    kubectl scale --namespace=${test_namespace} deployment/${nginx_deployment} --replicas=2
    if ! timeout 1m bash -c -- "until test \$(kubectl get --namespace=${test_namespace} deployment/${nginx_deployment} -o jsonpath='{..spec.replicas}') -eq 1; do sleep 1; done";
    then
        kubectl describe --namespace=${test_namespace} pods -l "app.kubernetes.io/instance=${release_name}"
        kubectl describe --namespace=${test_namespace} deployments ${nginx_deployment}
        operator_logs
        exit 1
    fi

    # update CR to replicaCount=2 and verify the deployment
    # automatically scales up to 2 replicas.
    kubectl patch --namespace=${test_namespace} nginxes.helm.example.com nginx-sample -p '[{"op":"replace","path":"/spec/replicaCount","value":2}]' --type=json
    if ! timeout 1m bash -c -- "until test \$(kubectl get --namespace=${test_namespace} deployment/${nginx_deployment} -o jsonpath='{..spec.replicas}') -eq 2; do sleep 1; done";
    then
        kubectl describe --namespace=${test_namespace} pods -l "app.kubernetes.io/instance=${release_name}"
        kubectl describe --namespace=${test_namespace} deployments ${nginx_deployment}
        operator_logs
        exit 1
    fi

    kubectl delete --namespace=${test_namespace} -f config/samples/helm.example_v1alpha1_nginx.yaml --wait=true
    kubectl logs deployment/nginx-operator-controller-manager -c manager --namespace=${operator_namespace} | grep "Uninstalled release" | grep "${release_name}"
}

# create and build the operator
mkdir nginx-operator
pushd nginx-operator
log=$(operator-sdk init --plugins=helm.operator-sdk.io/v1 \
  --domain=com --group=helm.example --version=v1alpha1 --kind=Nginx \
  2>&1)
echo $log
if echo $log | grep -q "failed to generate RBAC rules"; then
    echo FAIL expected successful generation of RBAC rules
    exit 1
fi


sed -i".bak" -E -e 's/(FROM quay.io\/operator-framework\/helm-operator)(:.*)?/\1:dev/g' Dockerfile; rm -f Dockerfile.bak
make docker-build IMG="$DEST_IMAGE"

# If using a kind cluster, load the image into all nodes.
load_image_if_kind "$DEST_IMAGE"

docker pull "$METRICS_TEST_IMAGE"
# If using a kind cluster, load the metrics test image into all nodes.
load_image_if_kind "$METRICS_TEST_IMAGE"

OPERATORDIR="$(pwd)"

deploy_operator
trap_add 'remove_operator' EXIT
test_operator