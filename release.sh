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

GO_VER="1.10"
if ! go version | cut -d" " -f3 | grep -q "$GO_VER"; then
	echo "must compile binaries with Go compiler version v${GO_VER}+"
	exit 1
fi

# Detect whether versions in code were updated.
VER_FILE="version/version.go"
TOML_TMPL_FILE="internal/pkg/scaffold/gopkgtoml.go"
ANS_TOML_TMPL_FILE="internal/pkg/scaffold/ansible/gopkgtoml.go"
HELM_TOML_TMPL_FILE="internal/pkg/scaffold/helm/gopkgtoml.go"
CURR_VER_VER_FILE="$(sed -nr 's/Version = "(.+)"/\1/p' "$VER_FILE" | tr -d ' \t\n')"
CURR_VER_TMPL_FILE="$(sed -nr 's/.*".*v(.+)".*#osdk_version_annotation/v\1/p' "$TOML_TMPL_FILE" | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER_VER_FILE" || "$VER" != "$CURR_VER_TMPL_FILE" ]]; then
	echo "versions are not set correctly in $VER_FILE or $TOML_TMPL_FILE"
	exit 1
fi
CURR_VER_ANS_TMPL_FILE="$(sed -nr 's/.*".*v(.+)".*#osdk_version_annotation/v\1/p' "$ANS_TOML_TMPL_FILE" | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER_ANS_TMPL_FILE" ]]; then
	echo "versions are not set correctly in $ANS_TOML_TMPL_FILE"
	exit 1
fi
CURR_VER_HELM_TMPL_FILE="$(sed -nr 's/.*".*v(.+)".*#osdk_version_annotation/v\1/p' "$HELM_TOML_TMPL_FILE" | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER_HELM_TMPL_FILE" ]]; then
	echo "versions are not set correctly in $HELM_TOML_TMPL_FILE"
	exit 1
fi

git tag --sign --message "Operator SDK $VER" "$VER"

git verify-tag --verbose "$VER"

make release
