#!/bin/bash

FAILED="false"

# clone OLM
mkdir olm-test
cd olm-test
git clone https://github.com/operator-framework/operator-lifecycle-manager.git
cd operator-lifecycle-manager

# Get list of unique minor versions sorted highest to lowest
MINOR_ARRAY=($(git tag --list --sort=-version:refname "v0.*" | awk -F. '{ print $1 "." $2 "." }' | sort -ur))

# Get highest patch version for each unique minor version
# Loop through the latest 3 minor versions
VERSIONS_STRING=""
for i in {2..0}
do
    MINOR_VERSION=$(echo "${MINOR_ARRAY[i]}")
    PATCH_ARRAY=($(git tag --list --sort=-version:refname "$MINOR_VERSION*" | awk -F. '{ print $1 "." $2 "." $3 }' | sort -ur))
    VERSIONS_STRING+=$(echo " ${PATCH_ARRAY[0]}" | tr -d 'v')
done

VERSIONS_ARRAY=($(echo "$VERSIONS_STRING" | tr ' ' '\n'))

# clean up OLM because we are done with it now
cd ../../
rm -rf olm-test

# check Makefile OLM_VERSIONS
EXPECTED="OLM_VERSIONS =$VERSIONS_STRING"
ACTUAL=$(cat Makefile | grep "OLM_VERSIONS =")

if [[ $ACTUAL != $EXPECTED ]]
then
    echo -e "\nMakefile does not have the most up to date OLM release versions.\nEXPECTED: $EXPECTED | ACTUAL: $ACTUAL"
    FAILED="true"
fi

# check internal/bindata/olm/versions.go
AVAILABLE_VERSIONS=($(cat internal/bindata/olm/versions.go | awk '/var availableVersions/,/},\n}/' | tr -d '\t' | tr -d ' ":{},'))

ACTUAL=""
for i in {1..3}
do 
    ACTUAL+=" ${AVAILABLE_VERSIONS[i]}"
done

if [[ $ACTUAL != $VERSIONS_STRING ]]
then
    echo -e "\ninternal/bindata/olm/versions.go does not have the most up to date OLM release versions as availableVersions.\nEXPECTED: $VERSIONS_STRING | ACTUAL: $ACTUAL"
    FAILED="true"
fi

# check internal/testutils/olm.go
EXPECTED="OlmVersionForTestSuite = \"${VERSIONS_ARRAY[2]}\""
ACTUAL=$(cat internal/testutils/olm.go | grep "OlmVersionForTestSuite =" | tr -d '\t')

if [[ $ACTUAL != $EXPECTED ]]
then
    echo -e "\ninternal/testutils/olm.go does not have the most up to date OLM release versions.\nEXPECTED: $EXPECTED | ACTUAL: $ACTUAL"
    FAILED="true"
fi

# check docs - website/content/en/docs/overview/_index.md
EXPECTED="Currently, the officially supported OLM Versions are: ${VERSIONS_ARRAY[0]}, ${VERSIONS_ARRAY[1]}, and ${VERSIONS_ARRAY[2]}"
ACTUAL=$(cat website/content/en/docs/overview/_index.md | grep "Currently, the officially supported OLM Versions are:")

if [[ $ACTUAL != $EXPECTED ]]
then
    echo -e "\nDocs (website/content/en/docs/overview/_index.md) does not have the most up to date OLM release versions.\nEXPECTED: $EXPECTED | ACTUAL: $ACTUAL"
    FAILED="true"
fi

# State pass or fail result
if [[ $FAILED != "false" ]]
then
    echo -e "\nOLM Version Check - \033[0;31mFAILED\033[0m"
    exit 1
else
    echo -e "\nOLM Version Check - \033[0;32mPASSED\033[0m"
fi

