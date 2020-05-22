#!/usr/bin/env bash

set -e

source ./hack/lib/common.sh

header_text "Building the site and checking links"
docker volume create sdk-html
docker run --rm -v "$(pwd):/src" -v sdk-html:/target klakegg/hugo:0.70.0-ext-ubuntu -s website
docker run --rm -v sdk-html:/target mtlynch/htmlproofer /target --empty-alt-ignore --http-status-ignore 429
docker volume rm sdk-html
