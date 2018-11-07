#!/bin/bash

set -o nounset
set -o pipefail

source "hack/lib/test_lib.sh"

echo "Checking case of error messages..."
allfiles=$(listFiles)
output=$(grep -ERn 'Fatal(f)?\("[[:upper:]]|Error(f)?\("[[:upper:]]|Error\((err|nil), "[[:upper:]]|errors.New\("[[:upper:]]' $allfiles)
if [ -n "${output}" ]; then
  echo "Error messages in wrong case:"
  echo "${output}"
  exit 255
fi
