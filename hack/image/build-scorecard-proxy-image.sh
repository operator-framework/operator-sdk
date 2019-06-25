#!/usr/bin/env bash

set -eux

# build operator binary and base image
WD="$(dirname "$(pwd)")"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
  go build \
  -gcflags "all=-trimpath=${WD}" \
  -asmflags "all=-trimpath=${WD}" \
  -o images/scorecard-proxy/scorecard-proxy \
  images/scorecard-proxy/cmd/proxy/main.go
pushd images/scorecard-proxy
docker build -t "$1" .
popd
