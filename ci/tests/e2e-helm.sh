#!/usr/bin/env bash

source hack/lib/test_lib.sh

set -eux

component="osdk-helm-e2e"
eval IMAGE=$IMAGE_FORMAT
component="osdk-helm-e2e-hybrid"
eval IMAGE2=$IMAGE_FORMAT
ROOTDIR="$(pwd)"
GOTMP="$(mktemp -d -p $GOPATH/src)"
trap_add 'rm -rf $GOTMP' EXIT

mkdir -p $ROOTDIR/bin
export PATH=$ROOTDIR/bin:$PATH

if ! [ -x "$(command -v kubectl)" ]; then
    curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.14.2/bin/linux/amd64/kubectl && chmod +x kubectl && mv kubectl bin/
fi

if ! [ -x "$(command -v oc)" ]; then
    curl -Lo oc.tar.gz https://github.com/openshift/origin/releases/download/v3.11.0/openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit.tar.gz
    tar xvzOf oc.tar.gz openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit/oc > oc && chmod +x oc && mv oc bin/ && rm oc.tar.gz
fi

oc version

make install

deploy_operator() {
    kubectl create -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role.yaml"
    kubectl create -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl create -f "$OPERATORDIR/deploy/crds/helm_v1alpha1_nginx_crd.yaml"
    kubectl create -f "$OPERATORDIR/deploy/operator.yaml"
}

remove_operator() {
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/service_account.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/role.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/role_binding.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/crds/helm_v1alpha1_nginx_crd.yaml"
    kubectl delete --wait=true --ignore-not-found=true -f "$OPERATORDIR/deploy/operator.yaml"
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
if oc api-versions | grep openshift; then
    oc project default
    oc adm policy add-scc-to-user anyuid -z default
fi

# create and build the operator
pushd "$GOTMP"
operator-sdk new nginx-operator --api-version=helm.example.com/v1alpha1 --kind=Nginx --type=helm

pushd nginx-operator
cp deploy/operator.yaml deploy/operator-copy.yaml
sed -i "s|REPLACE_IMAGE|$IMAGE|g" deploy/operator.yaml

OPERATORDIR="$(pwd)"

deploy_operator
trap_add 'remove_operator' EXIT
test_operator
remove_operator

# the nginx-operator pods remain after the deployment is gone; wait until the pods are removed
if ! timeout 60s bash -c -- "until kubectl get pods -l name=nginx-operator |& grep \"No resources found\"; do sleep 2; done";
then
    echo FAIL: nginx-operator Deployment did not get garbage collected
    kubectl describe pods -l "app.kubernetes.io/instance=${release_name}"
    kubectl describe deployments ${nginx_deployment}
    kubectl logs deployment/nginx-operator
    exit 1
fi

cp deploy/operator-copy.yaml deploy/operator.yaml
sed -i "s|REPLACE_IMAGE|$IMAGE2|g" deploy/operator.yaml
deploy_operator
trap_add 'remove_operator' EXIT
test_operator
remove_operator

popd
popd
