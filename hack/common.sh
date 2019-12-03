#!/usr/bin/env bash
set -e

# Turn colors in this script off by setting the NO_COLOR variable in your
# environment to any value:
NO_COLOR=${NO_COLOR:-""}
if [ -z "$NO_COLOR" ]; then
  header=$'\e[1;33m'
  error=$'\e[0;31m'
  reset=$'\e[0m'
else
  header=''
  error=''
  reset=''
fi

function header_text {
  echo "$header$*$reset"
}

function error_text {
  echo "$error$*$reset"
}

function fetch_go_linter {
  header_text "Checking if golangci-lint is installed"
  if ! is_installed golangci-lint; then
    header_text "Installing golangci-lint"
    curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b $(go env GOPATH)/bin v1.21.0
  fi
}

function is_installed {
  if command -v $1 &>/dev/null; then
    return 0
  fi
  return 1
}
