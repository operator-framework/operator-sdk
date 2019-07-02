#!/usr/bin/env bash

set -ex


test_version() {
    local version="$1"
    # If version is "latest", run without --version flag
    local ver_flag="--version=${version}"
    if [[ "$version" == "latest" ]]; then
      ver_flag=""
    fi

    # Status should fail with OLM not installed
    commandoutput=$(operator-sdk alpha olm status 2>&1 || true)
    echo $commandoutput | grep -F "Failed to get OLM status: no existing installation found"

    # Uninstall should fail with OLM not installed
    commandoutput=$(operator-sdk alpha olm uninstall 2>&1 || true)
    echo $commandoutput | grep -F "Failed to uninstall OLM: no existing installation found"

    # Install should succeed with nothing installed
    commandoutput=$(operator-sdk alpha olm install $ver_flag 2>&1)
    echo $commandoutput | grep -F "Successfully installed OLM version \\\"${version}\\\""

    # Install should fail with OLM Installed
    commandoutput=$(operator-sdk alpha olm install $ver_flag 2>&1 || true)
    echo $commandoutput | grep -F "Failed to install OLM version \\\"${version}\\\": detected existing OLM resources: OLM must be completely uninstalled before installation"

    # Status should succeed with OLM installed
    commandoutput=$(operator-sdk alpha olm status 2>&1)
    echo $commandoutput | grep -F "Successfully got OLM status"

    # Uninstall should succeed with OLM installed
    commandoutput=$(operator-sdk alpha olm uninstall 2>&1)
    echo $commandoutput | grep -F "Successfully uninstalled OLM"
}

test_version "latest"
test_version "0.10.1"
