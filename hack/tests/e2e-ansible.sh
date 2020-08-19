#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

source ./hack/lib/test_lib.sh
source ./hack/lib/image_lib.sh

# install SDK binaries
make install

# ansible proxy test require a running cluster; run during e2e instead
go test -count=1 ./internal/ansible/proxy/...

# create test directories
test_dir=./test
tests=$test_dir/e2e-ansible

export TRACE=1
export GO111MODULE=on

# set default envvars
setup_envs $tmp_sdk_root

go test $tests -v -ginkgo.v
