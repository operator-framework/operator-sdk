#!/usr/bin/env bash

filename="olm-bindata-"$1.go
echo $filename

function get_olm_manifests() {
    mkdir olm-manifests
    echo "downloading olm manifests for version ${1}"
    wget -O olm-manifests/olm.yaml "https://github.com/operator-framework/operator-lifecycle-manager/releases/download/${1}/olm.yaml"
    wget -O olm-manifests/crds.yaml "https://github.com/operator-framework/operator-lifecycle-manager/releases/download/${1}/crds.yaml"
}

function remove_olm_manifests {
    rm -rf olm-manifests
}

# check for files starting with the name "olm-bindata" inside internal/olm folder
function delete_old_olmbindata {
    echo "Deleting previous versions of olm-bindata files if they exist"
    find internal/olm -maxdepth 1 -type f -name olm-bindata-* -exec rm {} \;
}

FILE=internal/olm/$filename
if [ ! -f "$FILE" ]; then
    delete_old_olmbindata
    get_olm_manifests $1

    go get -u github.com/go-bindata/go-bindata/...
    $(go env GOPATH)/bin/go-bindata -o olm-bindata-$1.go -pkg olm olm-manifests/
    mv olm-bindata-$1.go internal/olm/

    remove_olm_manifests
fi

go mod tidy
