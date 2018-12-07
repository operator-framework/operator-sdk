#!/usr/bin/env bash

function log() { printf '%s\n' "$*"; }
function error() { log "ERROR: $*" >&2; }
function fatal() { error "$@"; exit 1; }
