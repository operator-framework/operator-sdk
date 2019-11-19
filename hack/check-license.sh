#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

source "hack/lib/test_lib.sh"

echo "Checking for license header..."
allfiles=$(listFiles)
licRes=""
for file in $allfiles; do
  if ! head -n3 "${file}" | grep -Eq "(Copyright|generated|GENERATED)" ; then
    licRes="${licRes}\n"$(echo -e "  ${file}")
  fi
done
if [ -n "${licRes}" ]; then
  echo -e "license header checking failed:\n${licRes}"
  exit 255
fi
