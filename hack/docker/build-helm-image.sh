#!/usr/bin/env bash

set -ex

# build operator binary and base image
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o test/helm-operator/helm-operator test/helm-operator/cmd/helm-operator/main.go
pushd test/helm-operator
docker build -t "$1" .
popd
