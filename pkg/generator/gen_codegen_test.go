// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generator

import (
	"bytes"
	"testing"
)

const boilerplateExp = `
`

const updateGeneratedExp = `#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

DOCKER_REPO_ROOT="/go/src/github.com/example-inc/app-operator"
IMAGE=${IMAGE:-"gcr.io/coreos-k8s-scale-testing/codegen:1.9.3"}

docker run --rm \
  -v "$PWD":"$DOCKER_REPO_ROOT":Z \
  -w "$DOCKER_REPO_ROOT" \
  "$IMAGE" \
  "/go/src/k8s.io/code-generator/generate-groups.sh"  \
  "deepcopy" \
  "github.com/example-inc/app-operator/pkg/generated" \
  "github.com/example-inc/app-operator/pkg/apis" \
  "app:v1alpha1" \
  --go-header-file "./tmp/codegen/boilerplate.go.txt" \
  $@
`

func TestCodeGen(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderBoilerplateFile(buf, appProjectName); err != nil {
		t.Error(err)
		return
	}
	if boilerplateExp != buf.String() {
		t.Errorf(errorMessage, boilerplateExp, buf.String())
	}

	buf = &bytes.Buffer{}
	if err := renderUpdateGeneratedFile(buf, appRepoPath, appApiDirName, appVersion); err != nil {
		t.Error(err)
		return
	}
	if updateGeneratedExp != buf.String() {
		t.Errorf(errorMessage, updateGeneratedExp, buf.String())
	}
}
