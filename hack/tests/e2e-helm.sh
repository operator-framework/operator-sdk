#!/usr/bin/env bash

#===================================================================
# FUNCTION trap_add ()
#
# Purpose:  prepends a command to a trap
#
# - 1st arg:  code to add
# - remaining args:  names of traps to modify
#
# Example:  trap_add 'echo "in trap DEBUG"' DEBUG
#
# See: http://stackoverflow.com/questions/3338030/multiple-bash-traps-for-the-same-signal
#===================================================================
trap_add() {
    trap_add_cmd=$1; shift || fatal "${FUNCNAME} usage error"
    new_cmd=
    for trap_add_name in "$@"; do
        # Grab the currently defined trap commands for this trap
        existing_cmd=`trap -p "${trap_add_name}" |  awk -F"'" '{print $2}'`

        # Define default command
        [ -z "${existing_cmd}" ] && existing_cmd="echo exiting @ `date`"

        # Generate the new command
        new_cmd="${trap_add_cmd};${existing_cmd}"

        # Assign the test
         trap   "${new_cmd}" "${trap_add_name}" || \
                fatal "unable to add to trap ${trap_add_name}"
    done
}

DEST_IMAGE="quay.io/example/nginx-operator:v0.0.2"

set -ex

# switch to the "default" namespace if on openshift
if which oc 2>/dev/null; then oc project default; fi

# build operator binary and base image
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o test/helm-operator/helm-operator test/helm-operator/cmd/helm-operator/main.go
pushd test/helm-operator
docker build -t quay.io/water-hole/helm-operator .
popd

# Make a test directory for Helm tests so we avoid using default GOPATH.
# Save test directory so we can delete it on exit.
HELM_TEST_DIR="$(mktemp -d)"
trap_add 'rm -rf $HELM_TEST_DIR' EXIT
cp -a test/helm-* "$HELM_TEST_DIR"
pushd "$HELM_TEST_DIR"

# Helm tests should not run in a Golang environment.
unset GOPATH GOROOT

# create and build the operator
operator-sdk new nginx-operator --api-version=helm.example.com/v1alpha1 --kind=Nginx --type=helm
pushd nginx-operator

operator-sdk build "$DEST_IMAGE"
sed -i "s|REPLACE_IMAGE|$DEST_IMAGE|g" deploy/operator.yaml
sed -i 's|Always|Never|g' deploy/operator.yaml
sed -i 's|size: 3|replicaCount: 1|g' deploy/crds/helm_v1alpha1_nginx_cr.yaml

DIR2="$(pwd)"
# deploy the operator
kubectl create -f deploy/service_account.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/service_account.yaml' EXIT
kubectl create -f deploy/role.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/role.yaml' EXIT
kubectl create -f deploy/role_binding.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/role_binding.yaml' EXIT
kubectl create -f deploy/crds/helm_v1alpha1_nginx_crd.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/crds/helm_v1alpha1_nginx_crd.yaml' EXIT
kubectl create -f deploy/operator.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/operator.yaml' EXIT

# wait for operator pod to run
if ! timeout 1m kubectl rollout status deployment/nginx-operator;
then
    kubectl logs deployment/nginx-operator
    exit 1
fi

# create CR
kubectl create -f deploy/crds/helm_v1alpha1_nginx_cr.yaml
trap_add 'kubectl delete --ignore-not-found -f ${DIR2}/deploy/crds/helm_v1alpha1_nginx_cr.yaml' EXIT
if ! timeout 1m bash -c -- 'until kubectl get nginxes.helm.example.com example-nginx -o jsonpath="{..status.release.info.status.code}" | grep 1; do sleep 1; done';
then
    kubectl logs deployment/nginx-operator
    exit 1
fi

release_name=$(kubectl get nginxes.helm.example.com example-nginx -o jsonpath="{..status.release.name}")
nginx_deployment=$(kubectl get deployment -l "app.kubernetes.io/instance=${release_name}" -o jsonpath="{..metadata.name}")

if ! timeout 1m kubectl rollout status deployment/${nginx_deployment};
then
    kubectl describe pods -l "app.kubernetes.io/instance=${release_name}"
    kubectl describe deployments ${nginx_deployment}
    kubectl logs deployment/${nginx_deployment}
    exit 1
fi

nginx_service=$(kubectl get service -l "app.kubernetes.io/instance=${release_name}" -o jsonpath="{..metadata.name}")
kubectl get service ${nginx_service}

kubectl delete -f deploy/crds/helm_v1alpha1_nginx_cr.yaml --wait=true
kubectl logs deployment/nginx-operator | grep "Uninstalled release" | grep "${release_name}"

popd
popd
