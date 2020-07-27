#!/usr/bin/env bash

set -eu

source hack/lib/test_lib.sh
source hack/lib/image_lib.sh
source ./hack/lib/common.sh

# ansible proxy test require a running cluster; run during e2e instead
go test -count=1 ./internal/ansible/proxy/...

DEST_IMAGE="quay.io/example/memcached-operator:v0.0.2"
ROOTDIR="$(pwd)"
TMPDIR="$(mktemp -d)"
trap_add 'rm -rf $TMPDIR' EXIT

setup_envs $tmp_sdk_root

deploy_operator() {
    header_text "Running deploy operator"
    IMG=$DEST_IMAGE make deploy
    kubectl create clusterrolebinding memcached-operator-system-metrics-reader --clusterrole=memcached-operator-metrics-reader --serviceaccount=default:default
}

remove_operator() {
    pushd $TMPDIR/memcached-operator
    header_text "Running remove operator"
    kubectl delete --ignore-not-found clusterrolebinding memcached-operator-system-metrics-reader
    make undeploy
    popd
}

operator_logs() {
    header_text "Getting Pod logs"
    kubectl describe pods
    header_text "Getting events"
    kubectl get events
    header_text "Getting operator logs"
    kubectl logs deployment/memcached-operator-controller-manager -c manager
}

test_operator() {
    header_text "Testing operator metrics"
    header_text "wait for operator pod to run"
    if ! timeout 1m kubectl rollout status deployment/memcached-operator-controller-manager;
    then
        error_text "FAIL: Failed to run"
        operator_logs
        exit 1
    fi

   header_text "verify that metrics service was created"
   if ! timeout 60s bash -c -- "until kubectl get service/memcached-operator-controller-manager-metrics-service > /dev/null 2>&1; do sleep 1; done";
   then
       error_text "FAIL: Failed to get metrics service"
       operator_logs
       exit 1
   fi

   header_text "verify that the metrics endpoint exists"
   serviceaccount_secret=$(kubectl get serviceaccounts default -n default -o jsonpath='{.secrets[0].name}')
   token=$(kubectl get secret ${serviceaccount_secret} -n default -o jsonpath={.data.token} | base64 -d)

   # verify that the metrics endpoint exists
   if ! timeout 60s bash -c -- "until kubectl run --attach --rm --restart=Never --namespace=default test-metrics --image=${METRICS_TEST_IMAGE} -- -sfkH \"Authorization: Bearer ${token}\" https://memcached-operator-controller-manager-metrics-service:8443/metrics; do sleep 1; done";
   then
       error_text "Failed to verify that metrics endpoint exists"
       operator_logs
       exit 1
   fi

    header_text "create custom resource (Memcached CR)"
    kubectl create -f config/samples/ansible_v1alpha1_memcached.yaml
    if ! timeout 60s bash -c -- 'until kubectl get deployment -l app=memcached | grep memcached; do sleep 1; done';
    then
        error_text "FAIL: Failed to verify to create memcached Deployment"
        operator_logs
        exit 1
    fi

    header_text "Wait for Operator Pod"
    if ! timeout 60s bash -c -- "until kubectl get pod -l control-plane=controller-manager; do sleep 1; done"
    then
        error_text "FAIL: Operator pod does not exist."
        operator_logs
        exit 1
    fi

    header_text "Ensure no liveness probe fail events"
    # We can't directly hit the endpoint, which is not publicly exposed. If k8s sees a failing endpoint, it will create a "Killing" event.
    live_pod=$(kubectl get pod -l control-plane=controller-manager -o jsonpath="{..metadata.name}")
    if kubectl get events --field-selector involvedObject.name=$live_pod | grep Killing
    then
        error_text "FAIL: Operator pod killed due to failed liveness probe."
        kubectl get events --field-selector involvedObject.name=$live_pod,reason=Killing
        operator_logs
        exit 1
    fi

    header_text "Verify that a config map owned by the CR has been created."
    if ! timeout 1m bash -c -- "until kubectl get configmap test-blacklist-watches > /dev/null 2>&1; do sleep 1; done";
    then
        error_text "FAIL: Unable to retrieve config map test-blacklist-watches."
        operator_logs
        exit 1
    fi

    header_text "Verify that config map requests skip the cache."
    if ! kubectl logs deployment/memcached-operator-controller-manager -c manager | grep -e "Skipping cache lookup\".*"Path\":\"\/api\/v1\/namespaces\/default\/configmaps\/test-blacklist-watches\";
    then
        error_text "FAIL: test-blacklist-watches should not be accessible with the cache."
        operator_logs
        exit 1
    fi


    header_text "verify that metrics reflect cr creation"
    if ! timeout 60s bash -c -- "until kubectl run --attach --rm --restart=Never --namespace=default test-metrics --image=${METRICS_TEST_IMAGE} -- -sfkH \"Authorization: Bearer ${token}\" https://memcached-operator-controller-manager-metrics-service:8443/metrics | grep memcached-sample; do sleep 1; done";
    then
        error_text "Failed to verify that metrics reflect cr creation"
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
    kubectl delete -f ${OPERATORDIR}/config/samples/ansible_v1alpha1_memcached.yaml --wait=true
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
    if kubectl logs deployment/memcached-operator-controller-manager -c manager| grep -i error;
    then
        error_text "FAIL: the operator log includes errors"
        operator_logs
        exit 1
    fi
}

header_text "Creating and building the operator"
pushd "$TMPDIR"
mkdir memcached-operator
pushd memcached-operator
operator-sdk init --plugins ansible.sdk.operatorframework.io/v1 \
  --domain example.com \
  --group ansible \
  --version v1alpha1 \
  --kind Memcached \
  --generate-playbook \
  --generate-role
cp "$ROOTDIR/test/ansible-memcached/tasks.yml" roles/memcached/tasks/main.yml
cp "$ROOTDIR/test/ansible-memcached/defaults.yml" roles/memcached/defaults/main.yml
cp -a "$ROOTDIR/test/ansible-memcached/memfin" roles/
marker=$(tail -n1 watches.yaml)
sed -i'.bak' -e '$ d' watches.yaml;rm -f watches.yaml.bak
cat "$ROOTDIR/test/ansible-memcached/watches-finalizer.yaml" >> watches.yaml
echo $marker >> watches.yaml
header_text "Adding a second Kind to test watching multiple GVKs"
operator-sdk create api --kind=Foo --group ansible --version=v1alpha1
sed -i".bak" -e 's/# FIXME.*/role: \/dev\/null/g' watches.yaml;rm -f watches.yaml.bak

sed -i".bak" -E -e 's/(FROM quay.io\/operator-framework\/ansible-operator)(:.*)?/\1:dev/g' Dockerfile; rm -f Dockerfile.bak
IMG=$DEST_IMAGE make docker-build
# If using a kind cluster, load the image into all nodes.
load_image_if_kind "$DEST_IMAGE"
make kustomize
if [ -f ./bin/kustomize ] ; then
  KUSTOMIZE="$(realpath ./bin/kustomize)"
else
  KUSTOMIZE="$(which kustomize)"
fi
pushd config/default
${KUSTOMIZE} edit set namespace default
popd

# kind has an issue with certain image registries (ex. redhat's), so use a
# different test pod image.
METRICS_TEST_IMAGE="curlimages/curl:latest"
docker pull "$METRICS_TEST_IMAGE"
# If using a kind cluster, load the metrics test image into all nodes.
load_image_if_kind "$METRICS_TEST_IMAGE"

OPERATORDIR="$(pwd)"

trap_add 'remove_operator' EXIT
deploy_operator
test_operator

popd
popd
