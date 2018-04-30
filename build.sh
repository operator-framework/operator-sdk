#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail



#build operator-sdk binary into ./bin
function operator_build{
...
}

#only build when called directly
if echo "$0" | grep "build$" >/dev/null; then
     operator_build
fi
