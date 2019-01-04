#!/usr/bin/env bash

DEST_IMAGE="quay.io/example/scorecard-proxy"

set -ex

# build scorecard-proxy image (and delete intermediate builder image)
docker build -t scorecard-proxy -f images/scorecard-proxy/Dockerfile .

# the test framework directory has all the manifests needed to run the cluster
pushd test/test-framework
commandoutput="$(operator-sdk scorecard --cr-manifest deploy/crds/cache_v1alpha1_memcached_cr.yaml --basic-tests --init-timeout 60 --olm-tests --csv-path deploy/memcachedoperator.0.0.2.csv.yaml --verbose 2>&1)"
echo $commandoutput
echo $commandoutput | grep "Total Score: 6/8 points"
popd
