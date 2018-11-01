function listPkgs() {
	go list ./commands/... ./pkg/... ./test/... | grep -v generated
}

function listFiles() {
	# make it work with composed gopath
	for gopath in ${GOPATH//:/ }; do
		if [[ "$(pwd)" =~ "$gopath" ]]; then
			GOPATH="$gopath"
			break
		fi
	done
	# pipeline is much faster than for loop
	listPkgs | xargs -I {} find "${GOPATH}/src/{}" -name '*.go' | grep -v generated
}
