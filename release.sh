#!/usr/bin/env bash

set -e

[ $# == 1 ] || { echo "usage: $0 version" && exit 1; }

VER=$1

[[ "$VER" =~ ^v[[:digit:]]+\.[[:digit:]]+\.[[:digit:]]+$ ]] || {
	echo "malformed version: \"$VER\""
	exit 2
}

if test -n "$(git ls-files --others | \
	grep --invert-match '\(vendor\|build/operator-sdk-v.\+\)')";
then
	echo "directory has untracked files"
	exit 1
fi

if ! $(git diff-index --quiet HEAD --); then
	echo "directory has uncommitted files"
	exit 1
fi

# Detect whether versions in code were updated.
CURR_VER="$(git describe --dirty --tags)"
VER_FILE="version/version.go"
TOML_TMPL_FILE="pkg/scaffold/gopkgtoml.go"
CURR_VER_VER_FILE="$(sed -nr 's/Version = "(.+)"/\1/p' "$VER_FILE" | tr -d '\s\t\n')"
CURR_VER_TMPL_FILE="$(sed -nr 's/.*".*v(.+)".*#osdk_version_annotation/v\1/p' "$TOML_TMPL_FILE" | tr -d '\s\t\n')"
if [ "$VER" != "$CURR_VER_VER_FILE" ] \
	|| [ "$VER" != "$CURR_VER_TMPL_FILE" ]; then
	echo "versions are not set correctly in $VER_FILE or $TOML_TMPL_FILE"
	exit 1
fi

git tag --sign --message "Operator SDK $VER" "$VER"

git verify-tag --verbose "$VER"

make release
