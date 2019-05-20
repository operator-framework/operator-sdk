#!/usr/bin/env bash

source hack/lib/test_lib.sh

DEST_IMAGE="quay.io/example/scorecard-proxy"
CONFIG_PATH=".test-osdk-scorecard.yaml"

set -ex

# build scorecard-proxy image
./hack/image/build-scorecard-proxy-image.sh "$DEST_IMAGE"

go build -o test/test-framework/scorecard/assets/osdk-scorecard-basic cmd/scorecard-basic/main.go
trap_add 'rm test/test-framework/scorecard/assets/osdk-scorecard-basic' EXIT
go build -o test/test-framework/scorecard/assets/osdk-scorecard-olm cmd/scorecard-olm/main.go
trap_add 'rm test/test-framework/scorecard/assets/osdk-scorecard-olm' EXIT

# the test framework directory has all the manifests needed to run the cluster
pushd test/test-framework
commandoutput="$(operator-sdk scorecard 2>&1)"
echo $commandoutput | grep "Total Score: 82%"

# test config file
commandoutput2="$(operator-sdk scorecard --config "$CONFIG_PATH")"
# check basic suite
echo $commandoutput2 | grep '^.*"error": 0,[[:space:]]"pass": 3,[[:space:]]"partialPass": 0,[[:space:]]"fail": 0,[[:space:]]"totalTests": 3,[[:space:]]"totalScorePercent": 100,.*$'
# check olm suite
echo $commandoutput2 | grep '^.*"error": 0,[[:space:]]"pass": 2,[[:space:]]"partialPass": 3,[[:space:]]"fail": 0,[[:space:]]"totalTests": 5,[[:space:]]"totalScorePercent": 74,.*$'
# check custom json result
echo $commandoutput2 | grep '^.*"error": 0,[[:space:]]"pass": 1,[[:space:]]"partialPass": 1,[[:space:]]"fail": 0,[[:space:]]"totalTests": 2,[[:space:]]"totalScorePercent": 71,.*$'
popd
