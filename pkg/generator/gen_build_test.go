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

const buildExp = `#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if ! which go > /dev/null; then
	echo "golang needs to be installed"
	exit 1
fi

BIN_DIR="$(pwd)/tmp/_output/bin"
mkdir -p ${BIN_DIR}
PROJECT_NAME="app-operator"
REPO_PATH="github.com/example-inc/app-operator"
BUILD_PATH="${REPO_PATH}/cmd/${PROJECT_NAME}"
echo "building "${PROJECT_NAME}"..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${BIN_DIR}/${PROJECT_NAME} $BUILD_PATH
`

const dockerFileExp = `FROM alpine:3.6

RUN adduser -D app-operator
USER app-operator

ADD tmp/_output/bin/app-operator /usr/local/bin/app-operator
`

func TestGenBuild(t *testing.T) {
	buf := &bytes.Buffer{}
	if err := renderBuildFile(buf, appRepoPath, appProjectName); err != nil {
		t.Error(err)
		return
	}
	if buildExp != buf.String() {
		t.Errorf(errorMessage, buildExp, buf.String())
	}

	buf = &bytes.Buffer{}
	if err := renderDockerBuildFile(buf); err != nil {
		t.Error(err)
		return
	}
	if dockerBuildTmpl != buf.String() {
		t.Errorf(errorMessage, dockerBuildTmpl, buf.String())
	}

	buf = &bytes.Buffer{}
	if err := renderDockerFile(buf, appProjectName); err != nil {
		t.Error(err)
		return
	}
	if dockerFileExp != buf.String() {
		t.Errorf(errorMessage, dockerFileExp, buf.String())
	}
}
