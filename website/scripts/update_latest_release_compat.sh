#!/usr/bin/env bash

# This script updates the latest release version's `kube_version` and `client_go_version`
# variable to be up to date. This change should be committed in the prerelease commit.

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
CONFIG_PATH="${DIR}/../config.toml"

KUBE_VERSION="$(cat Makefile | grep "export K8S_VERSION" | awk -F= '{ gsub(/ /,""); print $2 }')"
CLIENT_GO_VERSION="$(cat go.mod | grep "k8s.io/client-go" | awk -F" " '{ print $2 }')"

KUBE_MARKER="##LATEST_RELEASE_KUBE_VERSION##"
CLIENT_GO_MARKER="##LATEST_RELEASE_CLIENT_GO_VERSION##"

perl -0777 -pi -e $'s@'"${KUBE_MARKER}\\n  kube_version = ".+'@'"${KUBE_MARKER}\\n  kube_version = \"${KUBE_VERSION}\""'@g' ${CONFIG_PATH}
perl -0777 -pi -e $'s@'"${CLIENT_GO_MARKER}\\n  client_go_version = ".+'@'"${CLIENT_GO_MARKER}\\n  client_go_version = \"${CLIENT_GO_VERSION}\""'@g' ${CONFIG_PATH}