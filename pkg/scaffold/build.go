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

package scaffold

import (
	"io"

	"text/template"
)

type build struct {
	in *BuildInput
}

func NewbuildCodegen(in *BuildInput) Codegen {
	return &build{in: in}
}

type BuildInput struct {
	// ProjectName is the name of the operator project.
	ProjectName string
	// ProjectPath is the project path rooted at GOPATH. e.g "github.com/example/app-operator".
	ProjectPath string
}

func (d *build) Render(w io.Writer) error {
	t := template.New("build.go")
	t, err := t.Parse(buildTmpl)
	if err != nil {
		return err
	}

	return t.Execute(w, d.in)
}

const buildTmpl = `#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if ! which go > /dev/null; then
	echo "golang needs to be installed"
	exit 1
fi

BIN_DIR="$(pwd)/build/_output/bin"
mkdir -p ${BIN_DIR}
PROJECT_NAME={{.ProjectName}}
REPO_PATH={{.ProjectPath}}
BUILD_PATH="${REPO_PATH}/cmd/manager"
echo "building "${PROJECT_NAME}"..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${BIN_DIR}/${PROJECT_NAME} $BUILD_PATH
`
