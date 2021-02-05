#!/usr/bin/env bash

set -e

source ./hack/lib/common.sh

header_text "Building the website"
docker volume create sdk-html
trap_add "docker volume rm sdk-html" EXIT
docker run --rm -v "$(pwd):/src" -v sdk-html:/src/website/public klakegg/hugo:0.73.0-ext-ubuntu -s website

header_text "Checking links"
# For config explanation: https://github.com/gjtorikian/html-proofer#special-cases-for-the-command-line
docker run --rm -v sdk-html:/target klakegg/html-proofer:latest /target \
  --empty-alt-ignore \
  --http-status-ignore 429 \
  --allow_hash_href \
  --url-ignore "/github.com\/operator-framework\/operator-sdk\/edit\/master\//"
