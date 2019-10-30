#!/usr/bin/env bash

DEST_IMAGE="quay.io/example/scorecard-proxy"
CONFIG_PATH=".test-osdk-scorecard.yaml"
CONFIG_PATH_V1ALPHA1=".test-osdk-scorecard-v1alpha1.yaml"
CONFIG_PATH_DISABLE=".osdk-scorecard-disable.yaml"
CONFIG_PATH_INVALID=".osdk-scorecard-invalid.yaml"
CONFIG_PATH_V1ALPHA2=".osdk-scorecard-v1alpha2.yaml"

set -ex

# build scorecard-proxy image
./hack/image/build-scorecard-proxy-image.sh "$DEST_IMAGE"

# the test framework directory has all the manifests needed to run the cluster
pushd test/test-framework

# test to see if scorecard fails when version is v1alpha2 and when external plugins are configured
operator-sdk scorecard --version v1alpha2 --config "$CONFIG_PATH" |& grep '^.*error validating plugin config.*$'

# test to see if v1alpha2 is used from the command line
commandoutput="$(operator-sdk scorecard --version v1alpha2 --config "$CONFIG_PATH_V1ALPHA2" 2>&1)"
failCount=`echo $commandoutput | grep -o ": fail" | wc -l`
expectedFailCount=3
if [ $failCount -ne $expectedFailCount ]
then
	echo "expected fail count $expectedFailCount, got $failCount"
	exit 1
fi

# test to see if version in config file allows v1alpha1 to be specified
commandoutput="$(operator-sdk scorecard --config "$CONFIG_PATH_V1ALPHA1" 2>&1)"
echo $commandoutput | grep "Total Score: 67%"

commandoutput="$(operator-sdk scorecard --version v1alpha1 --config "$CONFIG_PATH" 2>&1)"
echo $commandoutput | grep "Total Score: 67%"

# test json output and default config path
commandoutput2="$(operator-sdk scorecard --version v1alpha1 2>&1)"
# check basic suite
echo $commandoutput2 | grep '^.*"error": 0,[[:space:]]"pass": 3,[[:space:]]"partialPass": 0,[[:space:]]"fail": 0,[[:space:]]"totalTests": 3,[[:space:]]"totalScorePercent": 100,.*$'
# check olm suite
echo $commandoutput2 | grep '^.*"error": 0,[[:space:]]"pass": 2,[[:space:]]"partialPass": 3,[[:space:]]"fail": 0,[[:space:]]"totalTests": 5,[[:space:]]"totalScorePercent": 74,.*$'
# check custom json result
echo $commandoutput2 | grep '^.*"error": 0,[[:space:]]"pass": 1,[[:space:]]"partialPass": 1,[[:space:]]"fail": 0,[[:space:]]"totalTests": 2,[[:space:]]"totalScorePercent": 71,.*$'
# check external no args
echo $commandoutput2 | grep '^.*"name": "Empty",[[:space:]]"description": "Test plugin with no args",[[:space:]]"earnedPoints": 2,[[:space:]]"maximumPoints": 3,.*$'
# check external flag
echo $commandoutput2 | grep '^.*"name": "Flags",[[:space:]]"description": "Test plugin with kubeconfig set via flags",[[:space:]]"earnedPoints": 2,[[:space:]]"maximumPoints": 4,.*$'
# check external env
echo $commandoutput2 | grep '^.*"name": "Environment",[[:space:]]"description": "Test plugin with kubeconfig set via env var",[[:space:]]"earnedPoints": 2,[[:space:]]"maximumPoints": 5,.*$'

# test kubeconfig flag (kubeconfig shouldn't exist so internal plugins should instantly fail)
commandoutput3="$(operator-sdk scorecard --version v1alpha1 --kubeconfig=/kubeconfig 2>&1)"
# check basic suite
echo $commandoutput3 | grep '^.*"name": "Basic Tests",[[:space:]]"description": "",[[:space:]]"error": 1,.*$'
# check olm suite
echo $commandoutput3 | grep '^.*"name": "OLM Integration",[[:space:]]"description": "",[[:space:]]"error": 1,.*$'
# check custom json result
echo $commandoutput3 | grep '^.*"error": 0,[[:space:]]"pass": 1,[[:space:]]"partialPass": 1,[[:space:]]"fail": 0,[[:space:]]"totalTests": 2,[[:space:]]"totalScorePercent": 71,.*$'
# check external no args
echo $commandoutput3 | grep '^.*"name": "Different Env",[[:space:]]"description": "Test plugin with /kubeconfig set via env var",[[:space:]]"earnedPoints": 3,[[:space:]]"maximumPoints": 3,.*$'
# check external flag
echo $commandoutput3 | grep '^.*"name": "Different Env and flag",[[:space:]]"description": "Test plugin with /kubeconfig set via env var and flag set",[[:space:]]"earnedPoints": 3,[[:space:]]"maximumPoints": 4,.*$'
# check external env (kubeconfig set in plugin config should override flag)
echo $commandoutput3 | grep '^.*"name": "Environment",[[:space:]]"description": "Test plugin with kubeconfig set via env var",[[:space:]]"earnedPoints": 2,[[:space:]]"maximumPoints": 5,.*$'

# Test invalid config
operator-sdk scorecard --version v1alpha1 --config "$CONFIG_PATH_INVALID" |& grep '^.*invalid keys.*$'


popd
