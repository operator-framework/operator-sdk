#!/usr/bin/env bash

#  Copyright 2020 The Operator-SDK Authors
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

set -eu

source ./hack/lib/test_lib.sh
source ./hack/lib/image_lib.sh

# Set versions
K8S_VERSION=v1.18.2
ETCD_VERSION=v3.4.3

# install SDK binaries
make install
export TRACE=1
export GO111MODULE=on

cleanup() {
  ! make uninstall
  ! make undeploy
}

###########################################################
# Project Quick-Start
###########################################################

# init the project
mkdir $GOPATH/src/quickstart-operator
pushd $GOPATH/src/quickstart-operator
trap_add 'rm -rf $GOPATH/src/quickstart-operator' EXIT

# Init project
cd $GOPATH/src/quickstart-operator
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

# Build the operator image
make docker-build  IMG=operator-sdk/quickstart-operator:0.0.1

# Load image on Kind cluster
load_image_if_kind operator-sdk/quickstart-operator:0.0.1

# Deploy operator
make deploy  IMG=operator-sdk/quickstart-operator:0.0.1

# Always, the default operator namespace will be:
operator_namespace="example-system"

# Ensure that the operator was deployed
if ! timeout 1m kubectl rollout status deployment/quickstart-operator-controller-manager --namespace=${operator_namespace} ;
then
    error_text "Failed to rollout status"
    kubectl get all --namespace=${operator_namespace}
    kubectl get events --namespace=${operator_namespace}
    kubectl logs deployment/quickstart-operator-controller-manager -c manager --namespace=${operator_namespace}
    exit 1
fi

trap_add 'cleanup' EXIT