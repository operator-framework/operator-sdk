#!/usr/bin/env bash

set -e

# No-op since the release:golang-1.12 base image currently has all required
# depedencies to build and test operator-sdk.
#
# TODO: pre-fetch modules here with `make tidy` and figure out permissions
#       issues in unit and sanity tests
# make tidy


