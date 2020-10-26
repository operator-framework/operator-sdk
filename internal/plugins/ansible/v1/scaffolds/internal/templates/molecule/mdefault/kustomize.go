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

var _ file.Template = &Kustomize{}

// Kustomize scaffolds a Kustomize for building a main
type Kustomize struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *Kustomize) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("molecule", "default", "kustomize.yml")
	}
	f.TemplateBody = kustomizeTemplate
	return nil
}

const kustomizeTemplate = `---
- name: Build kustomize testing overlay
  # load_restrictor must be set to none so we can load patch files from the default overlay
  command: '{{ "{{ kustomize }}" }} build  --load_restrictor none .'
  args:
    chdir: '{{ "{{ config_dir }}" }}/testing'
  register: resources
  changed_when: false

- name: Set resources to {{ "{{ state }}" }}
  k8s:
    definition: '{{ "{{ item }}" }}'
    state: '{{ "{{ state }}" }}'
    wait: yes
  loop: '{{ "{{ resources.stdout | from_yaml_all | list }}" }}'
`
