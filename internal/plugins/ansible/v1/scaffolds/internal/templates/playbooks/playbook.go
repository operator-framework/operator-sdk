// Copyright 2020 The Operator-SDK Authors
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

package playbooks

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

type Playbook struct {
	file.TemplateMixin
	file.ResourceMixin

	GenerateRole bool
}

func (f *Playbook) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("playbooks", "%[kind].yml")
		f.Path = f.Resource.Replacer().Replace(f.Path)
	}
	if f.Path == "" {
		f.Path = "playbook.yml"
	}
	f.TemplateBody = playbookTmpl
	return nil
}

const playbookTmpl = `---
- hosts: localhost
  gather_facts: no
  collections:
    - community.kubernetes
    - operator_sdk.util

  {{- if .GenerateRole }}
  tasks:
    - import_role:
        name: "{{.Resource.Kind | lower }}"
  {{- else }}
  tasks: []
	{{- end }}
`
