package generator

const boilerplateTmpl = `/*
Copyright YEAR The {{.ProjectName}} Authors

Commercial software license.
*/
`

const updateGeneratedTmpl = `#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

DOCKER_REPO_ROOT="/go/src/{{.RepoPath}}"
IMAGE=${IMAGE:-"gcr.io/coreos-k8s-scale-testing/codegen:1.9.3"}

docker run --rm \
  -v "$PWD":"$DOCKER_REPO_ROOT" \
  -w "$DOCKER_REPO_ROOT" \
  "$IMAGE" \
  "/go/src/k8s.io/code-generator/generate-groups.sh"  \
  "all" \
  "{{.RepoPath}}/pkg/generated" \
  "{{.RepoPath}}/pkg/apis" \
  "{{.APIDirName}}:{{.Version}}" \
  --go-header-file "./tmp/codegen/boilerplate.go.txt" \
  $@
`
