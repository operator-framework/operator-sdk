#!/usr/bin/env bash
set -eux

ROOTDIR="$(pwd)"
SCORECARD_DIR="/scorecard"

mkdir -p $SCORECARD_DIR

GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build \
  -gcflags "all=-trimpath=${ROOTDIR}" \
  -asmflags "all=-trimpath=${ROOTDIR}" \
  -o $SCORECARD_DIR/scorecard-proxy \
  images/scorecard-proxy/cmd/proxy/main.go
mv images/scorecard-proxy/bin $SCORECARD_DIR/bin
