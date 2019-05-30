#!/usr/bin/env bash
set -ex

go test ./test/e2e/... -root=$(pwd) -globalMan=test/e2e/testdata/empty.yaml -v -generate-only $1
