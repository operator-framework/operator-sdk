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
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

const WatchesYamlFile = "watches.yaml"

// WatchesYAML - watches yaml input wrapper
type WatchesYAML struct {
	input.Input

	Resource         scaffold.Resource
	GeneratePlaybook bool
	RolesDir         string
}

// GetInput - gets the input
func (s *WatchesYAML) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = WatchesYamlFile
	}
	s.RolesDir = RolesDir
	s.TemplateBody = watchesYAMLTmpl
	return s.Input, nil
}

const watchesYAMLTmpl = `---
- version: {{.Resource.Version}}
  group: {{.Resource.FullGroup}}
  kind: {{.Resource.Kind}}
{{ if .GeneratePlaybook }}  playbook: /opt/ansible/playbook.yaml{{ else }}  role: /opt/ansible/{{.RolesDir}}/{{.Resource.Kind}}{{ end }}
`
