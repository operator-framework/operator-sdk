#!/usr/bin/env bash

DEST_IMAGE="quay.io/example/scorecard-proxy"
CONFIG_PATH=".test-osdk-scorecard.yaml"
CONFIG_PATH_V1ALPHA1=".test-osdk-scorecard-v1alpha1.yaml"
CONFIG_PATH_DISABLE=".osdk-scorecard-disable.yaml"
CONFIG_PATH_INVALID=".osdk-scorecard-invalid.yaml"
CONFIG_PATH_V1ALPHA2=".osdk-scorecard-v1alpha2.yaml"
CONFIG_PATH_BUNDLE=".osdk-scorecard-bundle.yaml"

set -ex

source ./hack/lib/common.sh

# build scorecard-proxy image
./hack/image/build-scorecard-proxy-image.sh "$DEST_IMAGE"

# the test framework directory has all the manifests needed to run the cluster
pushd test/test-framework

header_text 'scorecard test to check if scorecard fails when version is v1alpha2 and external plugins are configured, return code 1 should be returned'
if ! commandoutput="$(operator-sdk scorecard --version v1alpha2 --config "$CONFIG_PATH" 2>&1)"; then
       	if ! (echo $commandoutput | grep -q '^.*error validating plugin config.*$'); then
		echo "expected scorecard to fail when version is v1alpha2 and when external plugins are configured"
		exit 1
	fi
else
	echo "test failed: expected return code 1"
	exit 1
fi

header_text 'scorecard test to see if bundle flag works correctly'
if ! commandoutput="$(operator-sdk scorecard --config "$CONFIG_PATH_BUNDLE" 2>&1)"; then 
	echo $commandoutput
	failCount=`echo $commandoutput | grep -o "fail" | wc -l`
	expectedFailCount=2
	if [ $failCount -ne $expectedFailCount ]
	then
		echo "expected fail count $expectedFailCount, got $failCount"
		exit 1
	fi
else
	echo "test failed: expected return code 1"
	exit 1
fi

header_text 'scorecard test to see if exit code 1 is returned on test failures'
if ! commandoutput="$(operator-sdk scorecard --version v1alpha2 --config "$CONFIG_PATH_V1ALPHA2" 2>&1)"; then 
	echo $commandoutput
	failCount=`echo $commandoutput | grep -o "fail" | wc -l`
	expectedFailCount=4
	if [ $failCount -ne $expectedFailCount ]
	then
		echo "expected fail count $expectedFailCount, got $failCount"
		exit 1
	fi
else
	echo "test failed: expected return code 1"
	exit 1
fi

header_text 'scorecard test to see if --list flag works'
commandoutput="$(operator-sdk scorecard --version v1alpha2 --list --selector=suite=basic --config "$CONFIG_PATH_V1ALPHA2" 2>&1)"
labelCount=`echo $commandoutput | grep -o "Label" | wc -l`
expectedLabelCount=3
if [ $labelCount -ne $expectedLabelCount ]
then
	echo "expected label count $expectedLabelCount, got $labelCount"
	exit 1
fi

header_text 'scorecard test to see if --selector flag works'
commandoutput="$(operator-sdk scorecard --version v1alpha2 --selector=suite=basic --config "$CONFIG_PATH_V1ALPHA2" 2>&1)"
labelCount=`echo $commandoutput | grep -o "Label" | wc -l`
expectedLabelCount=3
if [ $labelCount -ne $expectedLabelCount ]
then
	echo "expected label count $expectedLabelCount, got $labelCount"
	exit 1
fi

header_text 'scorecard test to see if version in config file allows v1alpha2 to be specified'
commandoutput="$(operator-sdk scorecard --config "$CONFIG_PATH_V1ALPHA1" 2>&1)"
echo $commandoutput | grep "Total Score: 64%"

header_text 'scorecard test to see if total score matches expected value'
commandoutput="$(operator-sdk scorecard --version v1alpha1 --config "$CONFIG_PATH" 2>&1)"
echo $commandoutput | grep "Total Score: 64%"

# test json output and default config path
commandoutput2="$(operator-sdk scorecard --version v1alpha1 2>&1)"

# check basic suite
header_text 'scorecard test to check basic suite expected scores'
echo $commandoutput2 | grep '^.*"error": 0,[[:space:]]"pass": 3,[[:space:]]"partialPass": 0,[[:space:]]"fail": 0,[[:space:]]"totalTests": 3,[[:space:]]"totalScorePercent": 100,.*$'

header_text 'scorecard test to check olm suite expected scores'
echo $commandoutput2 | grep '^.*"error": 0,[[:space:]]"pass": 1,[[:space:]]"partialPass": 3,[[:space:]]"fail": 1,[[:space:]]"totalTests": 5,[[:space:]]"totalScorePercent": 55,.*$'

header_text 'scorecard test to check custom json result'
echo $commandoutput2 | grep '^.*"error": 0,[[:space:]]"pass": 1,[[:space:]]"partialPass": 1,[[:space:]]"fail": 0,[[:space:]]"totalTests": 2,[[:space:]]"totalScorePercent": 71,.*$'

header_text 'scorecard test to check external no args'
echo $commandoutput2 | grep '^.*"name": "Empty",[[:space:]]"description": "Test plugin with no args",[[:space:]]"earnedPoints": 2,[[:space:]]"maximumPoints": 3,.*$'

header_text 'scorecard test to check external flag'
echo $commandoutput2 | grep '^.*"name": "Flags",[[:space:]]"description": "Test plugin with kubeconfig set via flags",[[:space:]]"earnedPoints": 2,[[:space:]]"maximumPoints": 4,.*$'

header_text 'scorecard test to check external env'
echo $commandoutput2 | grep '^.*"name": "Environment",[[:space:]]"description": "Test plugin with kubeconfig set via env var",[[:space:]]"earnedPoints": 2,[[:space:]]"maximumPoints": 5,.*$'

header_text 'scorecard test to test kubeconfig flag, kubeconfig should not exist so internal plugins should fail'
commandoutput3="$(operator-sdk scorecard --version v1alpha1 --kubeconfig=/kubeconfig 2>&1)"

header_text 'scorecard test to check basic suite'
echo $commandoutput3 | grep '^.*"name": "Basic Tests",[[:space:]]"description": "",[[:space:]]"error": 1,.*$'

header_text 'scorecard test to check olm suite'
echo $commandoutput3 | grep '^.*"name": "OLM Integration",[[:space:]]"description": "",[[:space:]]"error": 1,.*$'

header_text 'scorecard test to check custom json result'
echo $commandoutput3 | grep '^.*"error": 0,[[:space:]]"pass": 1,[[:space:]]"partialPass": 1,[[:space:]]"fail": 0,[[:space:]]"totalTests": 2,[[:space:]]"totalScorePercent": 71,.*$'

header_text 'scorecard test to check external no args'
echo $commandoutput3 | grep '^.*"name": "Different Env",[[:space:]]"description": "Test plugin with /kubeconfig set via env var",[[:space:]]"earnedPoints": 3,[[:space:]]"maximumPoints": 3,.*$'

header_text 'scorecard test to check external flag'
echo $commandoutput3 | grep '^.*"name": "Different Env and flag",[[:space:]]"description": "Test plugin with /kubeconfig set via env var and flag set",[[:space:]]"earnedPoints": 3,[[:space:]]"maximumPoints": 4,.*$'

header_text 'scorecard test to check external env, kubeconfig set in plugin config should override flag'
echo $commandoutput3 | grep '^.*"name": "Environment",[[:space:]]"description": "Test plugin with kubeconfig set via env var",[[:space:]]"earnedPoints": 2,[[:space:]]"maximumPoints": 5,.*$'

header_text 'scorecard test to check invalid config'
operator-sdk scorecard --version v1alpha1 --config "$CONFIG_PATH_INVALID" |& grep '^.*invalid keys.*$'


popd
