#!/usr/bin/env bash

set -e

source ./hack/lib/common.sh

header_text "Running markdown link checker against changed markdown files"

COMMIT_RANGE="$TRAVIS_COMMIT_RANGE"
if [ -z "$COMMIT_RANGE" ] || ! git rev-list --quiet $COMMIT_RANGE; then
	# Assume comparing current branch to master if we don't have $TRAVIS_COMMIT_RANGE
	COMMIT_RANGE="master"
fi

# todo: since the marker do not work with .website we will probably remove it
# However, if we decided to keep it then,
# - (by @joe): would be nice to be able to pass an explicit list of files to marker so we don't have to do these
# --exclude shenanigans. Also, note that we would need to fix the issue with the README.
# - The current version used of the marker is 0.8.0(latest) and it throws the error `received fatal alert: ProtocolVersion`
# becuase is not able supporting TLS 1.3 which is used in https://coreos.com/operators/. Then, it also need to be fixed.
find . -name '*.md' > .marker.tmp.all
git diff --name-only $COMMIT_RANGE | grep '\.md$' | sed -e 's/^/.\//' > .marker.tmp.changed
excludes=$(grep -vxF -f .marker.tmp.changed .marker.tmp.all | sed -e 's/^/--exclude /')
rm .marker.tmp.*

./hack/ci/marker --exclude website $excludes
