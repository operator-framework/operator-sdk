#!/usr/bin/env bash
set -e

source ./hack/common.sh

fetch_go_linter

header_text "Running golangci-lint"
golangci-lint run --disable-all \
    --deadline 5m \
    --enable=nakedret \
    --enable=structcheck \
    --enable=maligned \
    --enable=ineffassign \
    --enable=goconst \
    --enable=goimports \

##todo(camilamacedo86): The following checks requires fixes in the code
# --enable=golint
# --enable=gocyclo
# --enable=lll
# --enable=gosec
# --enable=errcheck \
# --enable=dupl \
# --enable=interfacer \
# --enable=misspell \
# --enable=varcheck \
# --enable=unparam \
