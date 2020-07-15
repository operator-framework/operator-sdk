#!/usr/bin/env bash

source hack/lib/common.sh

function listPkgDirs() {
	go list -f '{{.Dir}}' ./cmd/... ./pkg/... ./test/... ./internal/... | grep -v generated
}

function listFiles() {
	# pipeline is much faster than for loop
	listPkgDirs | xargs -I {} find {} -name '*.go' | grep -v generated
}

#===================================================================
# FUNCTION trap_add ()
#
# Purpose:  prepends a command to a trap
#
# - 1st arg:  code to add
# - remaining args:  names of traps to modify
#
# Example:  trap_add 'echo "in trap DEBUG"' DEBUG
#
# See: http://stackoverflow.com/questions/3338030/multiple-bash-traps-for-the-same-signal
#===================================================================
function trap_add() {
    trap_add_cmd=$1; shift || fatal "${FUNCNAME} usage error"
    new_cmd=
    for trap_add_name in "$@"; do
        # Grab the currently defined trap commands for this trap
        existing_cmd=`trap -p "${trap_add_name}" |  awk -F"'" '{print $2}'`

        # Define default command
        [ -z "${existing_cmd}" ] && existing_cmd="echo exiting @ `date`"

        # Generate the new command
        new_cmd="${trap_add_cmd};${existing_cmd}"

        # Assign the test
         trap   "${new_cmd}" "${trap_add_name}" || \
                fatal "unable to add to trap ${trap_add_name}"
    done
}

# check_dir accepts 3 args:
# 1: test case string
# 2: directory to test for existence
# 3: either 0 or 1, where 0 means "dir should exist", and 1 means
#   "dir should not exist". The command fails if the condition is not met.
function check_dir() {
  if [[ $3 == 0 ]]; then
    if [[ -d "$2" ]]; then
      error_text "${1}: directory ${2} should not exist"
      exit 1
    fi
  else
    if [[ ! -d "$2" ]]; then
      error_text "${1}: directory ${2} should exist"
      exit 1
    fi
  fi
}

# check_file accepts 3 args:
# 1: test case string
# 2: file to test for existence
# 3: either 0 or 1, where 0 means "file should exist", and 1 means
#   "file should not exist". The command fails if the condition is not met.
function check_file() {
  if [[ $3 == 0 ]]; then
    if [[ -f "$2" ]]; then
      error_text "${1}: file ${2} should not exist"
      exit 1
    fi
  else
    if [[ ! -f "$2" ]]; then
      error_text "${1}: file ${2} should exist"
      exit 1
    fi
  fi
}

function echo_run() {
	echo "$@"
	$@
}
