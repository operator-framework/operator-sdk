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

// Watches is the input needed to generate a deploy/crds/<group>_<version>_<kind>_crd.yaml file
type Watches struct {
	input.Input

	// Resource defines the inputs for the new watches yaml file
	Resource *Resource

	// PlaybookFile is the file name of the playbook yaml file
	PlaybookFile string
}

func (s *Watches) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = watchesYamlFile
	}
	s.PlaybookFile = playbookYamlFile
	s.TemplateBody = watchesTmpl
	return s.Input, nil
}

const watchesTmpl = `---
- version: {{ .Resource.Version }}
  group: {{ .Resource.GroupName }}
  kind: {{ .Resource.Kind }}
{{ if .GeneratePlaybook }}  playbook: /opt/ansible/{{ .PlaybookFile }}{{ else }}  role: /opt/ansible/roles/{{ .Resource.Kind }}{{ end }}
`
