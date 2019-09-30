#!/usr/bin/env bash

DEST_IMAGE="quay.io/example/scorecard-proxy"
CONFIG_PATH_V1ALPHA2=".osdk-scorecard-v1alpha2.yaml"

set -ex

# build scorecard-proxy image
./hack/image/build-scorecard-proxy-image.sh "$DEST_IMAGE"

# the test framework directory has all the manifests needed to run the cluster
pushd test/test-framework
commandoutput="$(operator-sdk scorecard --config "$CONFIG_PATH_V1ALPHA2" 2>&1)"
failCount=`echo $commandoutput | grep -o "fail" | wc -l`
if [ $failCount -ne 4 ]
then
	echo "expected fail count not equal to output"
	exit 1
fi

popd
