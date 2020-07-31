#!/usr/bin/env bash

set -eu

if [[ $# != 1 ]]; then
	echo "usage: $0 vX.Y.Z"
	exit 1
fi

VER=$1

NUMRE="0|[1-9][0-9]*"
PRERE="\-(alpha|beta|rc)\.[1-9][0-9]*"

if ! [[ "$VER" =~ ^v($NUMRE)\.($NUMRE)\.($NUMRE)($PRERE)?$ ]]; then
	echo "malformed version: \"$VER\""
	exit 1
fi

if git ls-files --others --exclude-standard | grep -Ev 'build/operator-sdk-v.+'; then
	echo "directory has untracked files"
	exit 1
fi

if ! git diff-index --quiet HEAD --; then
	echo "directory has uncommitted files"
	exit 1
fi

GO_VER="1.13"
if ! go version | cut -d" " -f3 | grep -q "$GO_VER"; then
	echo "must compile binaries with Go compiler version v${GO_VER}"
	exit 1
fi

# Detect whether versions in code were updated.
VER_FILE="internal/version/version.go"
CURR_VER="$(sed -nr 's|\s+Version\s+= "(.+)"|\1|p' "$VER_FILE" | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER" ]]; then
	echo "version is not set correctly in $VER_FILE"
	exit 1
fi

INSTALL_GUIDE_FILE="website/content/en/docs/installation/install-operator-sdk.md"
CURR_VER_INSTALL_GUIDE_FILE="$(sed -nr 's/.*RELEASE_VERSION=(.+)/\1/p' "$INSTALL_GUIDE_FILE" | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER_INSTALL_GUIDE_FILE" ]]; then
	echo "version '$VER' is not set correctly in $INSTALL_GUIDE_FILE"
	exit 1
fi

# Tag the release commit and verify its tag.
git tag --sign --message "Operator SDK $VER" "$VER"
git verify-tag --verbose "$VER"

# Run the release builds.
make release V=1

# Verify the signatures
for f in $(ls build/*.asc); do gpg --verify $f; done
