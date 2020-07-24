#!/usr/bin/env bash

set -eu

source ./hack/lib/test_lib.sh
source ./hack/lib/image_lib.sh

# Kubernetes version e.g v1.18.2
K8S_VERSION=v1.18.2
# ETCD version e.g v3.4.3
ETCD_VERSION=v3.4.3

# install SDK binaries
make install

export TRACE=1
export GO111MODULE=on

cleanup() {
  make uninstall
  make undeploy
}

# init the project
mkdir $GOPATH/src/example
pushd $GOPATH/src/example
trap_add 'rm -rf $GOPATH/src/example' EXIT

# init project
cd $GOPATH/src/example
operator-sdk init --domain my.domain

# Create an API
operator-sdk create api --group webapp --version v1 --kind Guestbook  --controller=true --resource=true --make=false

# Install the CRDs into the cluster:
make install

# Install Instances of Custom Resources
kubectl apply -f config/samples/webapp_v1_guestbook.yaml

# Configure env test: (specific for GO type built with SDK)
# More info: https://github.com/operator-framework/operator-sdk/issues/3461
curl -sSLo setup_envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/kubebuilder/master/scripts/setup_envtest_bins.sh
chmod +x setup_envtest.sh
./setup_envtest.sh ${K8S_VERSION} ${ETCD_VERSION}

make docker-build  IMG=operator-sdk/example:0.0.1
load_image_if_kind operator-sdk/example:0.0.1

make deploy  IMG=operator-sdk/example:0.0.1

# the default operator namespace will be:
operator_namespace="example-system"

# wait for operator pod to run
if ! timeout 1m kubectl rollout status deployment/example-controller-manager --namespace=${operator_namespace} ;
then
    error_text "Failed to rollout status"
    kubectl get all --namespace=${operator_namespace}
    kubectl get events --namespace=${operator_namespace}
    kubectl logs deployment/example-controller-manager -c manager --namespace=${operator_namespace}
    exit 1
fi

trap_add 'cleanup' EXIT