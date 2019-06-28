#!/usr/bin/env bash

set -ex

version=$(curl -sL https://api.github.com/repos/operator-framework/operator-lifecycle-manager/releases/latest | jq -r .tag_name)
if [[ -z "${version}" ]]; then
    echo "Could not determine latest version of OLM"
    exit 1
fi

# Status should fail with OLM not installed
commandoutput=$(operator-sdk alpha olm status 2>&1 || true)
echo $commandoutput | grep "Failed to get OLM status: no existing installation found"

# Reset should fail with OLM not installed
commandoutput=$(operator-sdk alpha olm uninstall 2>&1 || true)
echo $commandoutput | grep "Failed to uninstall OLM: failed to delete OLM version ${version}: no existing installation found"

# Init should succeed with nothing installed
commandoutput=$(operator-sdk alpha olm install 2>&1)
echo $commandoutput | grep "Successfully installed OLM version ${version}"

# Init should fail with OLM Installed
commandoutput=$(operator-sdk alpha olm install 2>&1 || true)
echo $commandoutput | grep "Failed to install OLM: detected existing OLM resources"

# Status should succeed with OLM installed (try with and without --version flag)
commandoutput=$(operator-sdk alpha olm status 2>&1)
echo $commandoutput | grep "Successfully got OLM status for version ${version}"

commandoutput=$(operator-sdk alpha olm status --version=${version} 2>&1)
echo $commandoutput | grep "Successfully got OLM status for version ${version}"

# Reset should succeed with OLM installed
commandoutput=$(operator-sdk alpha olm uninstall 2>&1)
echo $commandoutput | grep "Successfully uninstalled OLM version ${version}"
