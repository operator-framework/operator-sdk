#!/usr/bin/env bash
set -e

source ./hack/common.sh

fetch_go_linter

header_text "Running golangci-lint"
golangci-lint run --disable-all \
    --deadline 5m \
    --enable=nakedret \
    --enable=varcheck \
    --enable=deadcode \
    --enable=ineffassign \
    --enable=goconst \

##todo(camilamacedo86): The following checks requires fixes in the code
# --enable=golint
# --enable=gocyclo
# --enable=goimports
# --enable=lll
# --enable=gosec
# --enable=maligned
# --enable=errcheck \
# --enable=dupl \
# --enable=interfacer \
# --enable=misspell \
# --enable=structcheck \
# --enable=unparam \
