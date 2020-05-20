#!/usr/bin/env bash

set -e

source ./hack/lib/common.sh

header_text "Building the site and checking links"
docker run --rm -i -v "$(pwd):/src" -v "$(pwd)/website/public:/target" klakegg/hugo:0.70.0-ext-ubuntu -s website
docker run -v "$(pwd)/website/public:/target" mtlynch/htmlproofer /target --empty-alt-ignore --http-status-ignore 429
