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

# add_go_mod_replace adds a "replace" directive from $1 to $2 with an
# optional version version $3 to the current working directory's go.mod file.
function add_go_mod_replace() {
	local from_path="${1:?first path in replace statement is required}"
	local to_path="${2:?second path in replace statement is required}"
	local version="${3:-}"

	if [[ ! -d "$to_path" && -z "$version" ]]; then
		echo "second replace path $to_path requires a version be set because it is not a directory"
		exit 1
	fi
	if [[ ! -e go.mod ]]; then
		echo "go.mod file not found in $(pwd)"
		exit 1
	fi

	# Check if a replace line already exists. If it does, remove. If not, append.
	if grep -q "${from_path} =>" go.mod; then
		sed -E -i 's|^.+'"${from_path} =>"'.+$||g' go.mod
	fi
	# Do not use "go mod edit" so formatting stays the same.
	local replace="replace ${from_path} => ${to_path}"
	if [[ -n "$version" ]]; then
		replace="$replace $version"
	fi
	echo "$replace" >> go.mod
}
