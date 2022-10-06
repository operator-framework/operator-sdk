#!/usr/bin/env bash

base_version=0.16.1

function version_gt() {
    test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1";
}

function get_olm_manifests() {
    echo "downloading olm manifests for version ${1}"
    tag=$1
    if version_gt ${1} $base_version; then
        tag="v"$1
    fi
    echo "using the olm tag ${tag}"
    curl -L -o olm-manifests/$1-olm.yaml "https://github.com/operator-framework/operator-lifecycle-manager/releases/download/${tag}/olm.yaml"
    curl -L -o olm-manifests/$1-crds.yaml "https://github.com/operator-framework/operator-lifecycle-manager/releases/download/${tag}/crds.yaml"
}

function remove_olm_manifests {
    rm -rf olm-manifests
}

# check for files starting with the name "manifests" inside internal/bindata/olm folder
function delete_old_olmbindata {
    echo "Deleting previous versions of olm-bindata files if they exist"
    find internal/bindata/olm -maxdepth 1 -type f -name manifests* -exec rm {} \;
}

function generate_bindata() {
    $(go env GOPATH)/bin/go-bindata -o manifests.go -pkg olm olm-manifests/
    mv manifests.go internal/bindata/olm/

    remove_olm_manifests
}

# delete bindata if it already exists
delete_old_olmbindata

# get go-bindata tool
go install github.com/go-bindata/go-bindata/...@latest

mkdir olm-manifests
for v in $@; do
    echo "processing version $v"
    get_olm_manifests $v
done

generate_bindata
remove_olm_manifests

go mod tidy
