#!/usr/bin/env bash

set -e

# If running in Github actions: this should be set to "github.base_ref".
# If running in Travis CI: this should be set to "$TRAVIS_COMMIT_RANGE".
: ${1?"the first argument must be set to a commit-ish reference"}

# Patterns to ignore.
declare -a DOC_PATTERNS
DOC_PATTERNS=(
  "(\.md)"
  "(\.go.tmpl)"
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
  "^(\.github/workflows/)"
)

if ! git diff --name-only $1 | grep -qvE "$(IFS="|"; echo "${DOC_PATTERNS[*]}")"; then
  echo "true"
  exit 0
fi
