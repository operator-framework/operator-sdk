#!/bin/bash -ex

IMAGES=(scorecard-storage scorecard-untar)

for IMAGE in ${IMAGES[@]}; do
  DIGEST=$(tools/bin/digester "quay.io/operator-framework/${IMAGE}:latest")
  sed -i -E "s|(quay.io/operator-framework/${IMAGE}@)sha256:[0-9a-z]+|\1${DIGEST}|" internal/cmd/operator-sdk/scorecard/cmd.go internal/cmd/operator-sdk/scorecard/cmd_test.go
done
