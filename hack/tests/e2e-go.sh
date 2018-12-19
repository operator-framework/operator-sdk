#!/usr/bin/env bash
set -ex

go test -timeout 15m ./test/e2e/... -root=. -globalMan=testdata/empty.yaml $1
