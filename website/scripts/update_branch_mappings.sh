#!/usr/bin/env bash

# This script writes a branch-to-subdomain mapping for the previously created
# release branch to the hugo config. This change should be committed in the prerelease commit.

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
CONFIG_PATH="${DIR}/../config.toml"

VERSION="${1?"A Version is required"}"
VERSION_PATCHLESS="$(echo $VERSION | awk -F. '{ print v$1"."$2 }')"
VERSION_X_DOMAIN="$(echo $VERSION | awk -F. '{ print v$1"-"$2"-x" }')"

if grep -C 1 "\[\[params\.versions\]\]" website/config.toml | grep -q "version = \"${VERSION_PATCHLESS}\""; then
  echo "Version mapping ${VERSION_PATCHLESS} already exists, skipping"
  exit 0
fi

KUBE_VERSION="$(cat Makefile | grep "export K8S_VERSION" | awk -F= '{ gsub(/ /,""); print $2 }')"
CLIENT_GO_VERSION="$(cat go.mod | grep "k8s.io/client-go" | awk -F" " '{ print $2 }')"

MARKER="##RELEASE_ADDME##"
PARAMS_VERSION="[[params.versions]]\\n  version = \"${VERSION_PATCHLESS}\"\\n  url = \"https://${VERSION_X_DOMAIN}.sdk.operatorframework.io\"\\n  kube_version = \"${KUBE_VERSION}\"\\n  client_go_version = \"${CLIENT_GO_VERSION}\""

sed -i -E $'s@'${MARKER}'@'"${MARKER}\\n\\n${PARAMS_VERSION}"'@g' "$CONFIG_PATH"

# Ensure config.toml was updated.
if ! grep -q "url = \"https://${VERSION_X_DOMAIN}.sdk.operatorframework.io\"" "$CONFIG_PATH"; then
  echo "$0 failed to update config.toml"
  exit 1
fi
