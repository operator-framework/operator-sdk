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

package ansible

import (
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

//Dockerfile - docker file for creating image
type Dockerfile struct {
	input.Input

	GeneratePlaybook bool
	RolesDir         string
}

// GetInput - gets the input
func (d *Dockerfile) GetInput() (input.Input, error) {
	if d.Path == "" {
		d.Path = filepath.Join(scaffold.BuildDir, scaffold.DockerfileFile)
	}
	d.RolesDir = RolesDir
	d.TemplateBody = dockerFileAnsibleTmpl
	return d.Input, nil
}

const dockerFileAnsibleTmpl = `FROM quay.io/water-hole/ansible-operator

COPY {{.RolesDir}}/ ${HOME}/{{.RolesDir}}/
{{- if .GeneratePlaybook }}
COPY playbook.yaml ${HOME}/playbook.yaml{{ end }}
COPY watches.yaml ${HOME}/watches.yaml
`
