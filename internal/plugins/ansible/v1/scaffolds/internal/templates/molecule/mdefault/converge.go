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

package mdefault

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Converge{}

// Converge scaffolds a Converge for building a main
type Converge struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *Converge) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("molecule", "default", "converge.yml")
	}
	f.TemplateBody = convergeTemplate
	return nil
}

const convergeTemplate = `---
- name: Converge
  hosts: localhost
  connection: local
  gather_facts: no
  collections:
    - community.kubernetes

  tasks:
    - name: Create Namespace
      k8s:
        api_version: v1
        kind: Namespace
        name: '{{ "{{ namespace }}" }}'

    - import_tasks: kustomize.yml
      vars:
        state: present
`
