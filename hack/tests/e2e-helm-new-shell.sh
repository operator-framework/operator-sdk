#!/usr/bin/env bash

set -eux

source hack/lib/test_lib.sh
source hack/lib/image_lib.sh

DEST_IMAGE="quay.io/example/nginx-operator:v0.0.2"
TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT

test_namespace="nginx-operator-system"

remove_operator() {
    make undeploy
    kubectl delete --ignore-not-found namespace ${test_namespace}
}

test_operator() {
    # kind has an issue with certain image registries (ex. redhat's), so use a
    # different test pod image.
    local metrics_test_image="fedora:latest"

    # wait for operator pod to run
    if ! timeout 1m kubectl rollout status deployment.apps/nginx-operator-controller-manager -n ${test_namespace} ;
    then
        kubectl get events --namespace=${test_namespace}
        kubectl logs deployment.apps/nginx-operator-controller-manager -c manager --namespace=${test_namespace}
        exit 1
    fi

    # verify that metrics service was created
    if ! timeout 60s bash -c -- "until kubectl get service/nginx-operator-metrics --namespace=${test_namespace} > /dev/null 2>&1; do sleep 1; done";
    then
        echo "Failed to get metrics service"
        kubectl get events --namespace=${test_namespace}
        kubectl logs deployment.apps/nginx-operator-controller-manager -c manager --namespace=${test_namespace}
        exit 1
    fi


    # verify that the metrics endpoint exists
    if ! timeout 1m bash -c -- "until kubectl run --attach --rm --restart=Never test-metrics --image=${metrics_test_image} -- curl -sfo /dev/null http://nginx-operator-metrics:8383/metrics; do sleep 1; done";
    then
        echo "Failed to verify that metrics endpoint exists"
        kubectl get events --namespace=${test_namespace}
        kubectl logs deployment.apps/nginx-operator-controller-manager -c manager --namespace=${test_namespace}
        exit 1
    fi

    # create CR
    kubectl create --namespace=${test_namespace} -f config/samples/example_v1alpha1_nginx.yaml
    trap_add "kubectl delete --namespace=${test_namespace} --ignore-not-found -f ${OPERATORDIR}/config/samples/example_v1alpha1_nginx.yaml" EXIT
    if ! timeout 1m bash -c -- "until kubectl get --namespace=${test_namespace} Nginx nginx-sample -o jsonpath='{..status.deployedRelease.name}' | grep 'nginx-sample'; do sleep 1; done";
    then
        kubectl get events --namespace=${test_namespace}
        kubectl logs deployment.apps/nginx-operator-controller-manager -c manager --namespace=${test_namespace}
        exit 1
    fi

    # verify that the custom resource metrics endpoint exists
    if ! timeout 1m bash -c -- "until kubectl run --attach --rm --restart=Never test-cr-metrics --image=${metrics_test_image} -- curl -sfo /dev/null http://nginx-operator-metrics:8686/metrics; do sleep 1; done";
    then
        echo "Failed to verify that custom resource metrics endpoint exists"
        kubectl get events --namespace=${test_namespace}
        kubectl logs deployment.apps/nginx-operator-controller-manager -c manager --namespace=${test_namespace}
        exit 1
    fi

    header_text "verify that the servicemonitor is created"
    if ! timeout 1m bash -c -- "until kubectl get servicemonitors/nginx-operator-metrics --namespace=${test_namespace} > /dev/null 2>&1; do sleep 1; done";
    then
        error_text "FAIL: Failed to get service monitor"
        kubectl get events --namespace=${test_namespace}
        kubectl logs deployment.apps/nginx-operator-controller-manager -c manager --namespace=${test_namespace}
        exit 1
    fi

    release_name=$(kubectl get --namespace=${test_namespace} Nginx nginx-sample -o jsonpath="{..status.deployedRelease.name}")
    nginx_deployment=$(kubectl get --namespace=${test_namespace}  deployment.apps -l "app.kubernetes.io/instance=${release_name}" -o jsonpath="{..metadata.name}")

    if ! timeout 1m kubectl rollout --namespace=${test_namespace} status deployment.apps/${nginx_deployment};
    then
        kubectl get events --namespace=${test_namespace}
        kubectl describe --namespace=${test_namespace} pods -l "app.kubernetes.io/instance=${release_name}"
        kubectl describe --namespace=${test_namespace} deployments ${nginx_deployment}
        kubectl logs deployment.apps/nginx-operator-controller-manager -c manager
        exit 1
    fi

    nginx_service=$(kubectl get --namespace=${test_namespace} service -l "app.kubernetes.io/instance=${release_name}" -o jsonpath="{..metadata.name}")
    kubectl get --namespace=${test_namespace} service ${nginx_service}

    # scale deployment replicas to 2 and verify the
    # deployment automatically scales back down to 1.
    kubectl scale --namespace=${test_namespace}  deployment.apps/${nginx_deployment} --replicas=2
    if ! timeout 1m bash -c -- "until test \$(kubectl get --namespace=${test_namespace} deployment/${nginx_deployment} -o jsonpath='{..spec.replicas}') -eq 1; do sleep 1; done";
    then
        kubectl get events --namespace=${test_namespace}
        kubectl describe --namespace=${test_namespace} pods -l "app.kubernetes.io/instance=${release_name}"
        kubectl describe --namespace=${test_namespace}  deployment.apps ${nginx_deployment}
        kubectl logs deployment.apps/nginx-operator-controller-manager -c manager --namespace=${test_namespace}
        exit 1
    fi

    # update CR to replicaCount=2 and verify the deployment
    # automatically scales up to 2 replicas.
    kubectl patch --namespace=${test_namespace} nginxes.helm.example.com example-nginx -p '[{"op":"replace","path":"/spec/replicaCount","value":2}]' --type=json
    if ! timeout 1m bash -c -- "until test \$(kubectl get --namespace=${test_namespace} deployment/${nginx_deployment} -o jsonpath='{..spec.replicas}') -eq 2; do sleep 1; done";
    then
        kubectl get events --namespace=${test_namespace}
        kubectl describe --namespace=${test_namespace} pods -l "app.kubernetes.io/instance=${release_name}"
        kubectl describe --namespace=${test_namespace}  deployment.apps ${nginx_deployment}
        kubectl logs deployment.apps/nginx-operator-controller-manager -c manager --namespace=${test_namespace}
        exit 1
    fi

    kubectl delete --namespace=${test_namespace} -f config/samples/example_v1alpha1_nginx.yaml --wait=true
    kubectl logs deployment.apps/nginx-operator-controller-manager -c manager --namespace=${test_namespace} | grep "Uninstalled release" | grep "${release_name}"
}

# create and build the operator

mkdir nginx-operator
cd nginx-operator
log=$(operator-sdk init --plugins=helm.operator-sdk.io/v1 \
  --domain=com --group=example --version=v1alpha1 --kind=Nginx \
  2>&1)
echo $log
if echo $log | grep -q "failed to generate RBAC rules"; then
    echo FAIL expected successful generation of RBAC rules
    exit 1
fi

pushd ../nginx-operator
install_service_monitor_crd

sed -i".bak" -E -e 's/(FROM quay.io\/operator-framework\/helm-operator)(:.*)?/\1:dev/g' Dockerfile; rm -f Dockerfile.bak
make docker-build IMG="$DEST_IMAGE"

# If using a kind cluster, load the image into all nodes.
load_image_if_kind "$DEST_IMAGE"

make install
make deploy IMG="$DEST_IMAGE"

# kind has an issue with certain image registries (ex. redhat's), so use a
# different test pod image.
METRICS_TEST_IMAGE="fedora:latest"
docker pull "$METRICS_TEST_IMAGE"
# If using a kind cluster, load the metrics test image into all nodes.
load_image_if_kind "$METRICS_TEST_IMAGE"
OPERATORDIR="$(pwd)"

trap_add 'remove_operator' EXIT
test_operator

popd
popd
