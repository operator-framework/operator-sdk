#!/usr/bin/env bash
set -e

source ./hack/common.sh

fetch_go_linter

header_text "Running golangci-lint"
golangci-lint run --disable-all \
    --deadline 5m \
    --enable=nakedret \
    --enable=interfacer \
    --enable=maligned \
    --enable=ineffassign \
    --enable=goconst \
    --enable=goimports \
    --enable=errcheck \
    --enable=dupl \

##todo(camilamacedo86): The following checks requires fixes in the code
# --enable=golint
# --enable=gocyclo
# --enable=lll
# --enable=gosec
# --enable=misspell \
# --enable=varcheck \
# --enable=structcheck \
# --enable=unparam \
