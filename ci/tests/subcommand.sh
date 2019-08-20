#!/usr/bin/env bash
source hack/lib/test_lib.sh

set -eux

ROOTDIR="$(pwd)"
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

./hack/tests/test-subcommand.sh
./ci/tests/scorecard-subcommand.sh
# This can't run in openshift-ci because the test cluster already has OLM installed.
#./hack/tests/alpha-olm-subcommands.sh
