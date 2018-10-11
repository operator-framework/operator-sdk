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

type TestFrameworkDockerfile struct {
	input.Input

	// GoTestScriptPath is the test framework run script path pre-defined in go_test_script.go
	GoTestScriptPath string

	// GoTestScriptFile is the test framework run script file name
	GoTestScriptFile string

	// BuildOutputBinDir is the 'operator-sdk build' command binary output dir.
	BuildOutputBinDir string
}

func (s *TestFrameworkDockerfile) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(BuildTestDir, DockerfileFile)
	}

	goScriptInput, err := (&GoTestScript{}).GetInput()
	if err != nil {
		return input.Input{}, err
	}
	s.GoTestScriptPath = goScriptInput.Path
	s.GoTestScriptFile = GoTestScriptFile

	s.BuildOutputBinDir = BuildBinDir
	s.TemplateBody = testFrameworkDockerfileTmpl
	return s.Input, nil
}

const testFrameworkDockerfileTmpl = `ARG BASEIMAGE
FROM ${BASEIMAGE}

ADD {{ .BuildOutputBinDir }}/{{.ProjectName}}-test /usr/local/bin/{{.ProjectName}}-test
ARG NAMESPACEDMAN
ADD $NAMESPACEDMAN /namespaced.yaml
ADD {{ .GoTestScriptPath }} /{{ .GoTestScriptFile }}
`
