#!/usr/bin/env bash

set -e

source ./hack/lib/common.sh

header_text "Checking diff between ./doc and ./website..."

COMMIT_RANGE="$TRAVIS_COMMIT_RANGE"
if [ -z "$COMMIT_RANGE" ] || ! git rev-list --quiet $COMMIT_RANGE; then
	# Assume comparing current branch to master if we don't have $TRAVIS_COMMIT_RANGE
	COMMIT_RANGE="master"
fi

old_changes=$(git diff --name-only $COMMIT_RANGE -- ./doc | grep -v "^doc/proposals/" | wc -l)
new_changes=$(git diff --name-only $COMMIT_RANGE -- ./website | wc -l)

if [[ "$old_changes" -ne 0 && "$new_changes" -eq 0 ]]; then
  error_text "ERROR: Docs changes in ./doc not found in ./website"
  echo ""
  header_text "./doc changes:"
  echo "$(git diff --name-only $COMMIT_RANGE -- ./doc)"
  echo ""
  header_text "./website changes:"
  echo "$(git diff --name-only $COMMIT_RANGE -- ./website)"
  exit 1
fi

header_text "OK"
