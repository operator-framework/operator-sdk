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
GO_GOMOD="internal/pkg/scaffold/project/go_mod.go"
ANS_GOMOD="internal/pkg/scaffold/ansible/go_mod.go"
HELM_GOMOD="internal/pkg/scaffold/helm/go_mod.go"
CURR_VER="$(sed -nr 's|Version = "(.+)"|\1|p' "$VER_FILE" | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER" ]]; then
	echo "version is not set correctly in $VER_FILE"
	exit 1
fi
CURR_VER_GO="$(sed -E -n -r 's|.*github.com/operator-framework/operator-sdk ([^ \t\n]+).*|\1|p' "$GO_GOMOD" | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER_GO" ]]; then
	echo "version is not set correctly in $GO_GOMOD"
	exit 1
fi
CURR_VER_ANS="$(sed -E -n -r 's|.*github.com/operator-framework/operator-sdk ([^ \t\n]+).*|\1|p' "$ANS_GOMOD" | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER_ANS" ]]; then
	echo "version is not set correctly in $ANS_GOMOD"
	exit 1
fi
CURR_VER_HELM="$(sed -E -n -r 's|.*github.com/operator-framework/operator-sdk ([^ \t\n]+).*|\1|p' "$HELM_GOMOD" | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER_HELM" ]]; then
	echo "version is not set correctly in $HELM_GOMOD"
	exit 1
fi

git tag --sign --message "Operator SDK $VER" "$VER"

git verify-tag --verbose "$VER"

make release
