#!/usr/bin/env bash
set -ex

go test ./test/e2e/... -root=. -globalMan=testdata/empty.yaml $1
