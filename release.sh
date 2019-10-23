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

GO_VER="1.13"
if ! go version | cut -d" " -f3 | grep -q "$GO_VER"; then
	echo "must compile binaries with Go compiler version v${GO_VER}"
	exit 1
fi

# Detect whether versions in code were updated.
VER_FILE="version/version.go"
CURR_VER="$(sed -nr 's|\s+Version\s+= "(.+)"|\1|p' "$VER_FILE" | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER" ]]; then
	echo "version is not set correctly in $VER_FILE"
	exit 1
fi

GO_GOMOD="internal/scaffold/go_mod.go"
ANS_GOMOD="internal/scaffold/ansible/go_mod.go"
HELM_GOMOD="internal/scaffold/helm/go_mod.go"
CURR_VER_GO_GOMOD="$(sed -E -n -r 's|.*operator-sdk ([^ \t\n]+).*|\1|p' "$GO_GOMOD" | tail -1 | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER_GO_GOMOD" ]]; then
	echo "go.mod 'replace' entry version is not set correctly in $GO_GOMOD"
	exit 1
fi
CURR_VER_ANS_GOMOD="$(sed -E -n -r 's|.*operator-sdk ([^ \t\n]+).*|\1|p' "$ANS_GOMOD" | tail -1 | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER_ANS_GOMOD" ]]; then
	echo "go.mod 'replace' entry version is not set correctly in $ANS_GOMOD"
	exit 1
fi
CURR_VER_HELM_GOMOD="$(sed -E -n -r 's|.*operator-sdk ([^ \t\n]+).*|\1|p' "$HELM_GOMOD" | tail -1 | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER_HELM_GOMOD" ]]; then
	echo "go.mod 'replace' entry version is not set correctly in $HELM_GOMOD"
	exit 1
fi

INSTALL_GUIDE_FILE="doc/user/install-operator-sdk.md"
CURR_VER_INSTALL_GUIDE_FILE="$(sed -nr 's/.*RELEASE_VERSION=(.+)/\1/p' "$INSTALL_GUIDE_FILE" | tr -d ' \t\n')"
if [[ "$VER" != "$CURR_VER_INSTALL_GUIDE_FILE" ]]; then
	echo "version '$VER' is not set correctly in $INSTALL_GUIDE_FILE"
    exit 1
fi
git tag --sign --message "Operator SDK $VER" "$VER"

git verify-tag --verbose "$VER"

make release
