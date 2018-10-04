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
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

type Build struct {
	input.Input
}

func (s *Build) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(buildDir, buildFile)
	}
	s.TemplateBody = buildTmpl
	return s.Input, nil
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
REPO_PATH={{ .Repo }}
BUILD_PATH="${REPO_PATH}/cmd/manager"
echo "building "${PROJECT_NAME}"..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ${BIN_DIR}/${PROJECT_NAME} $BUILD_PATH
`
