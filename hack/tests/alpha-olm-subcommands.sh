#!/usr/bin/env bash

source hack/lib/test_lib.sh

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

function test_operator() {
  local tmp="$(mktemp -d)"
  trap_add "rm -rf $tmp" EXIT
  local operator_name="memcached-operator"
  local operator_version="0.0.3"
  local tf_deploy_dir="test/test-framework/deploy"
  cp -a "test/test-framework/deploy/olm-catalog/${operator_name}" "${tmp}/"
  find "${tmp}/${operator_name}/" -mindepth 1 -type d \
    -exec cp test/test-framework/deploy/crds/*_crd.yaml {} \;
  local manifests_dir="${tmp}/${operator_name}"
  local csv_name="${operator_name}.v${operator_version}"
  local commandoutput

  # down when no operator is up (should fail).
  commandoutput=$(operator-sdk alpha down olm "$manifests_dir" --operator-version "$operator_version" 2>&1 || true)
  echo $commandoutput | grep -F "Failed to uninstall operator: no operator with name \\\"${csv_name}\\\" is running"
  # up with manifests dir and version.
  commandoutput=$(operator-sdk alpha up olm "$manifests_dir" --operator-version "$operator_version" 2>&1)
  echo $commandoutput | grep -F "Successfully installed \\\"${csv_name}\\\""
  # up when the operator is already up (should fail).
  commandoutput=$(operator-sdk alpha up olm "$manifests_dir" --operator-version "$operator_version" 2>&1 || true)
  echo $commandoutput | grep -F "Failed to install operator: an operator with name \\\"${csv_name}\\\" is already running"
  # down with manifests dir and version.
  commandoutput=$(operator-sdk alpha down olm "$manifests_dir" --operator-version "$operator_version" 2>&1)
  echo $commandoutput | grep -F "Successfully uninstalled \\\"${csv_name}\\\""
}

set -ex

# olm install/uninstall
test_version "latest"
test_version "0.10.1"

# olm up/down
if ! operator-sdk alpha olm install; then
  echo "Failed to install OLM latest before 'olm up/down' test"
  exit 1
fi
test_operator
if ! operator-sdk alpha olm uninstall; then
  echo "Failed to uninstall OLM latest after 'olm up/down' test"
  exit 1
fi
