#!/usr/bin/env bash

set -e

# Make sure the TRAVIS_COMMIT_RANGE is valid, by catching any errors and exiting.
if [ -z "$TRAVIS_COMMIT_RANGE" ] || ! git rev-list --quiet $TRAVIS_COMMIT_RANGE; then
  echo "Invalid commit range. Skipping check for doc only update"
  return 0
fi

if ! git diff --name-only $TRAVIS_COMMIT_RANGE | grep -qvE '(\.md)|(\.MD)|(\.png)|(\.pdf)|^(doc/)|^(MAINTAINERS)|^(LICENSE)'; then
  echo "Only doc files were updated, not running the CI."
  exit 0
fi
