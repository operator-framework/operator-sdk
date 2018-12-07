#!/usr/bin/env bash

set -ex

# build operator binary and base image
go build -o test/ansible-operator/ansible-operator test/ansible-operator/cmd/ansible-operator/main.go
pushd test/ansible-operator
docker build -t "$1" .
popd
