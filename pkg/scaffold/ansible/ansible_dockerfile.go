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

type AnsibleDockerfile struct {
	input.Input

	// GeneratePlaybook indicates that a playbook should be copied into the
	// Ansible image if true.
	// Default is false.
	GeneratePlaybook bool

	// RolesDir is the roles directory for Ansible roles
	RolesDir string

	// PlaybookPath and PlaybookFile are the file path and name of the playbook yaml file
	PlaybookPath, PlaybookFile string

	// WatchesPath and WatchesFile are the file path and name of the watches yaml file
	WatchesPath, WatchesFile string
}

func (s *AnsibleDockerfile) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(buildDir, dockerfileFile)
	}
	s.RolesDir = rolesDir

	playbookInput, err := (&Playbook{}).GetInput()
	if err != nil {
		return input.Input{}, err
	}
	s.PlaybookPath = playbookInput.Path
	s.PlaybookFile = watchesYamlFile

	watchesInput, err := (&Watches{}).GetInput()
	if err != nil {
		return input.Input{}, err
	}
	s.WatchesPath = watchesInput.Path
	s.WatchesFile = watchesYamlFile

	s.TemplateBody = ansibleDockerfileTmpl
	return s.Input, nil
}

const ansibleDockerfileTmpl = `FROM quay.io/water-hole/ansible-operator

COPY {{ .RolesDir }} ${HOME}/roles/
{{- if .GeneratePlaybook }}
COPY {{ .PlaybookPath }} ${HOME}/{{ .PlaybookFile }}{{ end }}
COPY {{ .WatchesPath }} ${HOME}/{{ .WatchesFile }}
`
