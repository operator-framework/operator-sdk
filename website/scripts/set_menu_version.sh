#!/usr/bin/env bash

# This script updates the hugo config's "version_menu" param
# to the current ${MAJOR}.${MINOR} string.

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
CONFIG_PATH="${DIR}/../config.toml"

BRANCH_NAME="$(git rev-parse --abbrev-ref HEAD)"
if [[ "$BRANCH_NAME" =~ v[0-9]+\.[0-9]+\.x ]]; then
  VERSION_MENU="$(echo $BRANCH_NAME | awk -F. '{ print v$1"."$2 }')"
  sed -i -E 's/version_menu = ".+"/version_menu = "'${VERSION_MENU}'"/g' "$CONFIG_PATH"

  # Ensure config.toml was updated.
  if ! grep -q "version_menu = \"${VERSION_MENU}\"" "$CONFIG_PATH"; then
    echo "$0 failed to update config.toml"
    exit 1
  fi
fi
