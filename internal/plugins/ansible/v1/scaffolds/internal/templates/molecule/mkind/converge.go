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

package mkind

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
		f.Path = filepath.Join("molecule", "kind", "converge.yml")
	}
	f.TemplateBody = convergeTemplate
	return nil
}

const convergeTemplate = `---
- name: Converge
  hosts: localhost
  connection: local
  gather_facts: no

  tasks:
    - name: Build operator image
      docker_image:
        build:
          path: '{{ "{{ project_dir }}" }}'
          pull: no
        name: '{{ "{{ operator_image }}" }}'
        tag: latest
        push: no
        source: build
        force_source: yes

    - name: Load image into kind cluster
      command: kind load docker-image --name osdk-test '{{ "{{ operator_image }}" }}'
      register: result
      changed_when: '"not yet present" in result.stdout'

- import_playbook: ../default/converge.yml
`
