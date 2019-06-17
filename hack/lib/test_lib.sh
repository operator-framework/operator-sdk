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

function edit_replace_modfile() {
	local modfile="$1"
	local sdk_dir="$2"
	local sdk_repo="github.com/operator-framework/operator-sdk"

	# Remove any "replace" and "require" lines for the SDK repo before vendoring
	# in case this is a release PR and the tag doesn't exist yet. This must be
	# done without using "go mod edit", which first parses go.mod and will error
	# if it doesn't find a tag/version/package.
	# TODO: remove SDK repo references if PR/branch is not from the main SDK repo.
	sed -E -i 's|^.*'"$sdk_repo"'.*$||g' "$modfile"

	# Run "go mod vendor" to pull down the deps specified by the scaffolded
	# `go.mod` file.
	go mod vendor -v

	# Use the local operator-sdk directory as the repo. To make the go toolchain
	# happy, the directory needs a `go.mod` file that specifies the module name,
	# so we need this temporary hack until we update the SDK repo itself to use
	# go modules.
	echo "module ${sdk_repo}" > "${sdk_dir}/go.mod"
	trap_add "rm ${sdk_dir}/go.mod" EXIT
	go mod edit -replace="${sdk_repo}=${sdk_dir}"
	go mod vendor -v
}
