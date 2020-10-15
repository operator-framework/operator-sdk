#!/usr/bin/env bash

function get_olm_manifests() {
    echo "downloading olm manifests for version ${1}"
    wget -O olm-manifests/$1-olm.yaml "https://github.com/operator-framework/operator-lifecycle-manager/releases/download/${1}/olm.yaml"
    wget -O olm-manifests/$1-crds.yaml "https://github.com/operator-framework/operator-lifecycle-manager/releases/download/${1}/crds.yaml"
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
go get -u github.com/go-bindata/go-bindata/...

eval "arr=($1)"
mkdir olm-manifests
for v in "${arr[@]}"; do 
    echo "processing version $v"
    get_olm_manifests $v
done

generate_bindata
remove_olm_manifests

go mod tidy
