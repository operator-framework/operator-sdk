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

DEST_IMAGE="quay.io/example/memcached-operator:v0.0.2"

set -ex

# switch to the "default" namespace if on openshift, to match the minikube test
if which oc 2>/dev/null; then oc project default; fi

# build operator binary and base image
go build -o test/ansible-operator/ansible-operator test/ansible-operator/cmd/ansible-operator/main.go
pushd test
pushd ansible-operator
docker build -t quay.io/water-hole/ansible-operator .
popd

# get current directory so we can delete project on exit
DIR1=$(pwd)
trap_add 'rm -rf ${DIR1}/memcached-operator' EXIT
# create and build the operator
operator-sdk new memcached-operator --api-version=ansible.example.com/v1alpha1 --kind=Memcached --type=ansible
cp ansible-memcached/tasks.yml memcached-operator/roles/Memcached/tasks/main.yml
cp ansible-memcached/defaults.yml memcached-operator/roles/Memcached/defaults/main.yml
cp -a ansible-memcached/memfin memcached-operator/roles/
cat ansible-memcached/watches-finalizer.yaml >> memcached-operator/watches.yaml

pushd memcached-operator
operator-sdk build $DEST_IMAGE
sed -i "s|REPLACE_IMAGE|$DEST_IMAGE|g" deploy/operator.yaml
sed -i 's|Always|Never|g' deploy/operator.yaml

DIR2=$(pwd)
# deploy the operator
kubectl create -f deploy/service_account.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/service_account.yaml' EXIT
kubectl create -f deploy/role.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/role.yaml' EXIT
kubectl create -f deploy/role_binding.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/role_binding.yaml' EXIT
kubectl create -f deploy/crds/ansible_v1alpha1_memcached_crd.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/crds/ansible_v1alpha1_memcached_crd.yaml' EXIT
kubectl create -f deploy/operator.yaml
trap_add 'kubectl delete -f ${DIR2}/deploy/operator.yaml' EXIT

# wait for operator pod to run
if ! timeout 1m kubectl rollout status deployment/memcached-operator;
then
    kubectl logs deployment/memcached-operator
    exit 1
fi

# create CR
kubectl create -f deploy/crds/ansible_v1alpha1_memcached_cr.yaml
trap_add 'kubectl delete --ignore-not-found -f ${DIR2}/deploy/crds/ansible_v1alpha1_memcached_cr.yaml' EXIT
if ! timeout 20s bash -c -- 'until kubectl get deployment -l app=memcached | grep memcached; do sleep 1; done';
then
    kubectl logs deployment/memcached-operator
    exit 1
fi
memcached_deployment=$(kubectl get deployment -l app=memcached -o jsonpath="{..metadata.name}")
if ! timeout 1m kubectl rollout status deployment/${memcached_deployment};
then
    kubectl logs deployment/${memcached_deployment}
    exit 1
fi

kubectl delete -f ${DIR2}/deploy/crds/ansible_v1alpha1_memcached_cr.yaml --wait=true
kubectl logs deployment/memcached-operator | grep "this is a finalizer"

popd
popd
