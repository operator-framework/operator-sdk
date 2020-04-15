#!/bin/bash

source ./hack/lib/common.sh

ROOTDIR="$(pwd)"

# This function do scaffolds in the testdata/ for we are able to check/test its
# integration with kubebuilder
test_gen_kb_integration() {
    project=$1
    mkdir -p ./testdata/$project
    rm -rf ./testdata/$project/*
    pushd .
    cd ./testdata/$project

    sdk=operator-sdk
    header_text "Generating $project ..."

    export GO111MODULE=on
    export PATH=$PATH:$(go env GOPATH)/bin
    header_text "initializing $project..."

    $sdk init --domain=example.com
    if [ $project == "memcached-operator" ]; then
        header_text 'Creating APIs ...'
        $sdk create api --group=cache --version=v1alpha1 --kind=Memcached --controller=true --resource=true

        header_text 'Replacing controller example source-code ...'
        cp $ROOTDIR/example/kb-memcached-operator/memcached_controller.go.tmpl controllers/memcached_controller.go

        header_text 'Replacing api example source-code ...'
        cp $ROOTDIR/example/kb-memcached-operator/memcached_types.go.tmpl api/v1alpha1/memcached_types.go

        header_text 'Adding webhook'
        $sdk create webhook --group=cache --version=v1alpha1 --kind=Memcached --defaulting --programmatic-validation

        header_text 'Running sdk instructions ...'
        # todo: replace for operator-sdk generate k8s when it be available to the kube layout
        make generate

        header_text 'Remove go.sum ..'
        rm -f go.sum
    fi

    popd
}

set -e
make install
test_gen_kb_integration memcached-operator
