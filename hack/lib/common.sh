#!/usr/bin/env bash

function log() { printf '%s\n' "$*"; }
function error() { error_text "ERROR:" $* >&2; }
function fatal() { error "$@"; exit 1; }

# Turn colors in this script off by setting the NO_COLOR variable in your
# environment to any value:
NO_COLOR=${NO_COLOR:-""}
if [ -z "$NO_COLOR" ]; then
  header_color=$'\e[1;33m'
  error_color=$'\e[0;31m'
  reset_color=$'\e[0m'
else
  header_color=''
  error_color=''
  reset_color=''
fi

function header_text {
  echo "$header_color$*$reset_color"
}

function error_text {
  echo "$error_color$*$reset_color"
}

function is_installed {
  if command -v $1 &>/dev/null; then
    return 0
  fi
  return 1
}
