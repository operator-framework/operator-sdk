#!/usr/bin/env bash

set -e

# Make sure the TRAVIS_COMMIT_RANGE is valid, by catching any errors and exiting.
if [ -z "$TRAVIS_COMMIT_RANGE" ] || ! git rev-list --quiet $TRAVIS_COMMIT_RANGE; then
  echo "Invalid commit range. Skipping check for doc only update"
elif "$SKIP_DOC_CHECK" = "true"; then
  echo "Skipping check for doc only update. Env Var SKIP_DOC_CHECK = true"
elif ! git diff --name-only $TRAVIS_COMMIT_RANGE | grep -qvE '(\.md)|(\.MD)|(\.png)|(\.pdf)|^(doc/)|^(MAINTAINERS)|^(LICENSE)'; then
  echo "Only doc files were updated, not running the CI."
  exit 0
fi
