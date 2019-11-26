#!/usr/bin/env bash
set -e

source ./hack/common.sh

fetch_go_linter

header_text "Running golangci-lint"
golangci-lint run --disable-all \
    --deadline 5m \
    --enable=nakedret \
    --enable=ineffassign \
    --enable=goconst \
    --enable=lll \

##todo(camilamacedo86): The following checks requires fixes in the code
# --enable=golint
# --enable=gocyclo
# --enable=goimports
# --enable=lll
# --enable=gosec
# --enable=maligned
# --enable=deadcode \
# --enable=misspell \
# --enable=errcheck \
# --enable=dupl \
# --enable=interfacer \
# --enable=misspell \
# --enable=ineffassign \
# --enable=varcheck \
# --enable=structcheck \
# --enable=unparam \
