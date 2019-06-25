#!/usr/bin/env bash

set -e

export GOPROXY=https://proxy.golang.org/
make tidy
