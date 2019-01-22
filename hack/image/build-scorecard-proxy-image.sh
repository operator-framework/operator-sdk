#!/usr/bin/env bash

set -eux

# build operator binary and base image
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o images/scorecard-proxy/scorecard-proxy images/scorecard-proxy/cmd/proxy/main.go
pushd images/scorecard-proxy
docker build -t "$1" .
popd
