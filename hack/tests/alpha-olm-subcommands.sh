#!/usr/bin/env bash

set -ex


test_version() {
    local version="$1"

    # Status should fail with OLM not installed
    commandoutput=$(operator-sdk alpha olm status --version=${version} 2>&1 || true)
    echo $commandoutput | grep -F "Failed to get OLM status for version \\\"${version}\\\": no existing installation found"

    # Uninstall should fail with OLM not installed
    commandoutput=$(operator-sdk alpha olm uninstall --version=${version} 2>&1 || true)
    echo $commandoutput | grep -F "Failed to uninstall OLM version \\\"${version}\\\": no existing installation found"

    # Install should succeed with nothing installed
    commandoutput=$(operator-sdk alpha olm install --version=${version} 2>&1)
    echo $commandoutput | grep -F "Successfully installed OLM version \\\"${version}\\\""

    # Install should fail with OLM Installed
    commandoutput=$(operator-sdk alpha olm install --version=${version} 2>&1 || true)
    echo $commandoutput | grep -F "Failed to install OLM version \\\"${version}\\\": detected existing OLM resources: OLM must be completely uninstalled before installation"

    # Status should succeed with OLM installed
    # If version is "latest", also run without --version flag
    if [[ "$version" == "latest" ]]; then
        commandoutput=$(operator-sdk alpha olm status 2>&1)
        echo $commandoutput | grep -F "Successfully got OLM status for version \\\"${version}\\\""
    fi

    commandoutput=$(operator-sdk alpha olm status --version=${version} 2>&1)
    echo $commandoutput | grep -F "Successfully got OLM status for version \\\"${version}\\\""

    # Uninstall should succeed with OLM installed
    commandoutput=$(operator-sdk alpha olm uninstall --version=${version} 2>&1)
    echo $commandoutput | grep -F "Successfully uninstalled OLM version \\\"${version}\\\""
}

test_version "latest"
test_version "0.10.1"
