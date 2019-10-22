#!/usr/bin/env bash
source hack/lib/test_lib.sh
set -ex

# the test framework directory has all the manifests needed to run the cluster
pushd test/test-framework

ROOTDIR="$(pwd)"
CONFIG_PATH="$ROOTDIR/.test-osdk-scorecard-ci.yaml"
DEFAULT_CONFIG_PATH="$ROOTDIR/.osdk-scorecard.yaml"
DEFAULT_CONFIG_PATH_CI="$ROOTDIR/.osdk-scorecard-ci.yaml"
CONFIG_PATH_INVALID="$ROOTDIR/.osdk-scorecard-invalid.yaml"
component="scorecard-proxy"
eval IMAGE=$IMAGE_FORMAT

# we need to make the default config path have the CI config, which does not set the proxy pull policy to never
cp $DEFAULT_CONFIG_PATH $ROOTDIR/backup.yaml
cp $DEFAULT_CONFIG_PATH_CI $DEFAULT_CONFIG_PATH
trap_add 'cp $ROOTDIR/backup.yaml $DEFAULT_CONFIG_PATH && rm $ROOTDIR/backup.yaml' EXIT
sed 's|REPLACE_IMAGE|'$IMAGE'|g' -i $DEFAULT_CONFIG_PATH

cp $CONFIG_PATH $ROOTDIR/backup2.yaml
trap_add 'cp $ROOTDIR/backup2.yaml $CONFIG_PATH && rm $ROOTDIR/backup2.yaml' EXIT
sed 's|REPLACE_IMAGE|'$IMAGE'|g' -i $CONFIG_PATH

# basic test with specified config location
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
