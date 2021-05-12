#!/usr/bin/env bash

# This script updates the operator-sdk download link with the current release version.
# This change should be committed in the prerelease commit.

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
DOC_PATH="${DIR}/../content/en/docs/installation/_index.md"

VERSION="${1?"A Version is required"}"

TARGET="export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/"

sed -i -E 's@('"${TARGET}"').+@\1'"${VERSION}"'@g' "$DOC_PATH"

# Ensure the file was updated.
if ! grep -q "${TARGET}${VERSION}" "$DOC_PATH"; then
  echo "$0 failed to update ${DOC_PATH}"
  exit 1
fi
