#!/usr/bin/env bash

DEST_IMAGE="quay.io/example/scorecard-proxy"
CONFIG_PATH=".test-osdk-scorecard.yaml"
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

header_text 'scorecard test to test kubeconfig flag, kubeconfig should not exist so internal plugins should fail'
if ! commandoutput="$(operator-sdk scorecard --kubeconfig=/kubeconfig 2>&1)"; then 
	echo $commandoutput
else
	echo "test failed: expected return code 1"
	exit 1
fi

header_text 'scorecard test to see if total score matches expected value'
if ! commandoutput="$(operator-sdk scorecard --config "$CONFIG_PATH" 2>&1)"; then
	echo $commandoutput
	passCount=`echo $commandoutput | grep -o "pass" | wc -l`
	expectedPassCount=10
	if [ $passCount -ne $expectedPassCount ]
	then
		echo "expected label count $expectedLabelCount, got $labelCount"
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
	expectedFailCount=3
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
if ! commandoutput="$(operator-sdk scorecard --config "$CONFIG_PATH_V1ALPHA2" 2>&1)"; then 
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
commandoutput="$(operator-sdk scorecard --list --selector=suite=basic --config "$CONFIG_PATH_V1ALPHA2" 2>&1)"
labelCount=`echo $commandoutput | grep -o "Label" | wc -l`
expectedLabelCount=3
if [ $labelCount -ne $expectedLabelCount ]
then
	echo "expected label count $expectedLabelCount, got $labelCount"
	exit 1
fi

header_text 'scorecard test to see if --selector flag works'
commandoutput="$(operator-sdk scorecard --selector=suite=basic --config "$CONFIG_PATH_V1ALPHA2" 2>&1)"
labelCount=`echo $commandoutput | grep -o "Label" | wc -l`
expectedLabelCount=6
if [ $labelCount -ne $expectedLabelCount ]
then
	echo "expected label count $expectedLabelCount, got $labelCount"
	exit 1
fi

header_text 'scorecard test to check invalid config'
operator-sdk scorecard --config "$CONFIG_PATH_INVALID" |& grep '^.*invalid keys.*$'

popd
