#!/usr/bin/env bash

set -e

source ./hack/lib/common.sh

header_text "Running markdown link checker against changed markdown files"

COMMIT_RANGE="$TRAVIS_COMMIT_RANGE"
if [ -z "$COMMIT_RANGE" ] || ! git rev-list --quiet $COMMIT_RANGE; then
	# Assume comparing current branch to master if we don't have $TRAVIS_COMMIT_RANGE
	COMMIT_RANGE="master"
fi

find . -name '*.md' > .marker.tmp.all
git diff --name-only $COMMIT_RANGE | grep '\.md$' | sed -e 's/^/.\//' > .marker.tmp.changed
excludes=$(grep -vxF -f .marker.tmp.changed .marker.tmp.all | sed -e 's/^/--exclude /')
rm .marker.tmp.*

./hack/ci/marker --exclude website $excludes
