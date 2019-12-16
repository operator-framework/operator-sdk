#!/usr/bin/env bash
set -e

source ./hack/lib/common.sh

function fetch_go_linter {
  header_text "Checking if golangci-lint is installed"
  if ! is_installed golangci-lint; then
    header_text "Installing golangci-lint"
    curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b $(go env GOPATH)/bin v1.21.0
  fi
}

DEV_LINTERS=(
    ##todo(camilamacedo86): The following checks requires fixes in the code.
    ##todo(camilamacedo86): they should be enabled and added in the CI
    "--enable=gocyclo"
    "--enable=lll"
    "--enable=gosec"  # NOT add this one to CI since was defined that it should be optional for now at least.
)

subcommand=$1
case $subcommand in
	"dev")
	  ##todo(camilamacedo86): It should be removed when all linter checks be enabled
	  header_text "Checking the project with all linters (dev)"
		LINTERS=${DEV_LINTERS[@]}
		;;
	"ci")
	  header_text "Checking the project with the linters enabled for the ci"
		;;
	*)
		echo "Must pass 'dev' or 'ci' argument"
		exit 1
esac

fetch_go_linter

header_text "Running golangci-lint"
golangci-lint run --disable-all \
    --deadline 5m \
    --enable=nakedret \
    --enable=interfacer \
    --enable=varcheck \
    --enable=deadcode \
    --enable=structcheck \
    --enable=misspell \
    --enable=maligned \
    --enable=ineffassign \
    --enable=goconst \
    --enable=goimports \
    --enable=errcheck \
    --enable=dupl \
    --enable=unparam \
    --enable=golint \
    ${LINTERS[@]}
