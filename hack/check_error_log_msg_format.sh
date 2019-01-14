#!/bin/bash

set -o nounset
set -o pipefail

source "hack/lib/test_lib.sh"

echo "Checking format of error and log messages..."
allfiles=$(listFiles)
log_case_output=$(grep -PRn '(Error\((.*[Ee]rr|nil), |^(?!.*fmt).+\.Error(f)?\(|Fatal(f)?\(|Info(f)?\(|Warn(f)?\()"[[:lower:]]' $allfiles | sort -u)
if [ -n "${log_case_output}" ]; then
  echo -e "Log messages do not begin with upper case:\n${log_case_output}"
fi
err_case_output=$(grep -ERn '(errors\.New|fmt\.Errorf)\("[[:upper:]]' $allfiles | sort -u)
if [ -n "${err_case_output}" ]; then
  echo -e "Error messages do not begin with lower case:\n${err_case_output}"
fi
err_punct_output=$(grep -ERn '(errors\.New|fmt\.Errorf)\(".*\."' $allfiles | sort -u)
if [ -n "${err_punct_output}" ]; then
  echo -e "Error messages should not have ending punctuation:\n${err_punct_output}"
fi

if [[ -n "$log_case_output" || -n "$err_case_output" || -n "$err_punct_output" ]]; then
  exit 255
fi
