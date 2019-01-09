#!/bin/bash

set -o nounset
set -o pipefail

source "hack/lib/test_lib.sh"

echo "Checking case of error messages..."
allfiles=$(listFiles)
log_case_output=$(grep -ERn '(Error\((.*[Ee]rr|nil), |[^(fmt\.)]Error(f)?\(|Fatal(f)?\(|Info(f)?\(|Warn(f)?\()"[[:lower:]]' $allfiles)
if [ -n "${log_case_output}" ]; then
  echo "Log messages in wrong case:"
  echo "${log_case_output}"
fi
err_case_output=$(grep -ERn '(errors\.New|fmt\.Errorf)\("[[:upper:]]' $allfiles)
if [ -n "${err_case_output}" ]; then
  echo "Error messages in wrong case:"
  echo "${err_case_output}"
fi
err_punct_output=$(grep -ERn '(errors\.New|fmt\.Errorf)\(".*\."' $allfiles)
if [ -n "${err_punct_output}" ]; then
  echo "Error messages have ending punctuation:"
  echo "${err_punct_output}"
fi

if [[ -n "$log_case_output" || -n "$err_case_output" || -n "$err_punct_output" ]]; then
  exit 255
fi
