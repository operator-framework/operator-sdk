#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

DOCKER_REPO_ROOT="/go/src/github.com/coreos/play"
IMAGE=${IMAGE:-"gcr.io/coreos-k8s-scale-testing/codegen"}

docker run --rm \
  -v "$PWD":"$DOCKER_REPO_ROOT" \
  -w "$DOCKER_REPO_ROOT" \
  "$IMAGE" \
  "/go/src/k8s.io/code-generator/generate-groups.sh"  \
  "all" \
  "github.com/coreos/play/pkg/generated" \
  "github.com/coreos/play/pkg/apis" \
  "play:v1alpha1" \
  --go-header-file "./hack/codegen/boilerplate.go.txt" \
  $@
.