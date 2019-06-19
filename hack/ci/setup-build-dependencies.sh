#!/usr/bin/env bash

set -e

# Install dep
curl -Lo dep https://github.com/golang/dep/releases/download/v0.5.3/dep-linux-amd64 && chmod +x dep && mv dep /usr/local/bin/

# Ensure vendor directory is up-to-date
make dep
