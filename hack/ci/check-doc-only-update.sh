#!/usr/bin/env bash

set -e

# Make sure the TRAVIS_COMMIT_RANGE is valid, by catching any errors and exiting.
if [ -z "$TRAVIS_COMMIT_RANGE" ] || ! git rev-list --quiet $TRAVIS_COMMIT_RANGE; then
  echo "Invalid commit range. Skipping check for doc only update"
  return 0
fi

# Patterns to ignore.
declare -a DOC_PATTERNS
DOC_PATTERNS=(
  "(\.md)"
  "(\.MD)"
  "(\.png)"
  "(\.pdf)"
  "^(doc/)"
  "^(website/)"
  "^(changelog/)"
  "^(OWNERS)"
  "^(MAINTAINERS)"
  "^(SECURITY)"
  "^(LICENSE)"
)

if ! git diff --name-only $TRAVIS_COMMIT_RANGE | grep -qvE "$(IFS="|"; echo "${DOC_PATTERNS[*]}")"; then
  echo "Only doc files were updated, not running the CI."
  exit 0
fi
