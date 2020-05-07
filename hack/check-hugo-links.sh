#!/usr/bin/env bash

set -e

source ./hack/lib/common.sh

header_text "Building html and checking links"

pushd website
npm install postcss-cli autoprefixer
hugo
liche -d public -r -c 50 public
popd
