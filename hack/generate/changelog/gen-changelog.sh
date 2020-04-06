#!/bin/bash

source ./hack/lib/common.sh
set -e
shopt -s extglob

[[ -n "$GEN_CHANGELOG_TAG" ]] || fatal "Must set GEN_CHANGELOG_TAG (e.g. export GEN_CHANGELOG_TAG=v1.2.3)"
go run ./hack/generate/changelog/gen-changelog.go -tag="${GEN_CHANGELOG_TAG}"
rm ./changelog/fragments/!(00-template.yaml)
