#!/usr/bin/env bash
source hack/lib/test_lib.sh

set -ex

if [ -z "$KUBECONFIG" ]; then
  KUBECONFIG=$HOME/.kube/config
fi

# Create an app-operator project that defines the App CR.
mkdir -p $HOME/projects/example-inc/
# Create a new app-operator project
cd $HOME/projects/example-inc/
operator-sdk new app-operator --repo github.com/example-inc/app-operator
cd app-operator

# Add a new API for the custom resource AppService
operator-sdk add api --api-version=app.example.com/v1alpha1 --kind=AppService

# Add a new controller that watches for AppService
operator-sdk add controller --api-version=app.example.com/v1alpha1 --kind=AppService

# Build and push the app-operator image to a public registry such as quay.io
operator-sdk build quay.io/operator-framework/app-operator:dev

# Login to public registry such as quay.io
$ docker login quay.io

# Push image
$ docker push quay.io/operator-framework/app-operator:dev

# Update the operator manifest to use the built image name (if you are performing these steps on OSX, see note below)
$ sed -i "" 's|REPLACE_IMAGE|quay.io/operator-framework/app-operator:dev|g' deploy/operator.yaml

# Setup Service Account
$ kubectl create -f deploy/service_account.yaml
# Setup RBAC
$ kubectl create -f deploy/role.yaml
$ kubectl create -f deploy/role_binding.yaml
# Setup the CRD
$ kubectl create -f deploy/crds/app.example.com_appservices_crd.yaml
# Deploy the app-operator
$ kubectl create -f deploy/operator.yaml

# Create an AppService CR
# The default controller will watch for AppService objects and create a pod for each CR
$ kubectl create -f deploy/crds/app.example.com_v1alpha1_appservice_cr.yaml

#
$

header_text 'Verify that a pod is created'
if ! commandoutput="$(kubectl get pod -l app=example-appservice 2>&1)"; then
	echo $commandoutput
	failCount=`echo $commandoutput | grep -o "error" | wc -l`
	expectedFailCount=0
	if [ $failCount -ne $expectedFailCount ]
	then
		echo "expected fail count $expectedFailCount, got $failCount"
		exit 1
	fi
else
	echo "test failed: expected return code 1"
	exit 1
fi

header_text 'Test the new Resource Type'
if ! commandoutput="$(kubectl describe appservice example-appservice 2>&1)"; then
	echo $commandoutput
	failCount=`echo $commandoutput | grep -o "error" | wc -l`
	expectedFailCount=0
	if [ $failCount -ne $expectedFailCount ]
	then
		echo "expected fail count $expectedFailCount, got $failCount"
		exit 1
	fi
else
	echo "test failed: expected return code 1"
	exit 1
fi


# Delete
kubectl delete -f deploy/crds/app.example.com_v1alpha1_appservice_cr.yaml

# Local test
header_text 'Check if i running locally'
if ! commandoutput="$(operator-sdk run --local 2>&1)"; then
	echo $commandoutput
	failCount=`echo $commandoutput | grep -o "error" | wc -l`
	expectedFailCount=0
	if [ $failCount -ne $expectedFailCount ]
	then
		echo "expected fail count $expectedFailCount, got $failCount"
		exit 1
	fi
else
	echo "test failed: expected return code 1"
	exit 1
fi

# Cleanup
kubectl delete -f deploy/role.yaml
kubectl delete -f deploy/role_binding.yaml
kubectl delete -f deploy/service_account.yaml
kubectl delete -f deploy/crds/app.example.com_appservices_crd.yaml
kubectl delete -f deploy/operator.yaml