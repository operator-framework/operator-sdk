#!/usr/bin/env bash

DEST_IMAGE="quay.io/example/scorecard-proxy"
CONFIG_PATH=".test-osdk-scorecard.yaml"

set -ex

# build scorecard-proxy image (and delete intermediate builder image)
./hack/image/build-scorecard-proxy-image.sh "$DEST_IMAGE"

# the test framework directory has all the manifests needed to run the cluster
pushd test/test-framework
commandoutput="$(operator-sdk scorecard \
  --cr-manifest deploy/crds/cache_v1alpha1_memcached_cr.yaml \
  --init-timeout 60 \
  --csv-path deploy/memcachedoperator.0.0.2.csv.yaml \
  --verbose \
  --proxy-image "$DEST_IMAGE" \
  --proxy-pull-policy Never \
  2>&1)"
echo $commandoutput | grep "Total Score: 6/8 points"

# test config file
commandoutput2="$(operator-sdk scorecard \
  --proxy-image "$DEST_IMAGE" \
  --config "$CONFIG_PATH")"
echo $commandoutput2 | grep "Total Score: 6/8 points"
popd
