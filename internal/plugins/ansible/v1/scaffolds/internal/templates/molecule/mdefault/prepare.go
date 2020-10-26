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

var _ file.Template = &Prepare{}

// Prepare scaffolds a Prepare for building a main
type Prepare struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *Prepare) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("molecule", "default", "prepare.yml")
	}
	f.TemplateBody = prepareTemplate
	return nil
}

const prepareTemplate = `---
- name: Prepare
  hosts: localhost
  connection: local
  gather_facts: false

  tasks:
    - name: Ensure operator image is set
      fail:
        msg: |
          You must specify the OPERATOR_IMAGE environment variable in order to run the
          'default' scenario
      when: not operator_image

    - name: Set testing image
      command: '{{ "{{ kustomize }}" }} edit set image testing={{ "{{ operator_image }}" }}'
      args:
        chdir: '{{ "{{ config_dir }}" }}/testing'

    - name: Set pull policy
      command: '{{ "{{ kustomize }}" }} edit add patch pull_policy/{{ "{{ operator_pull_policy }}" }}.yaml'
      args:
        chdir: '{{ "{{ config_dir }}" }}/testing'

    - name: Set testing namespace
      command: '{{ "{{ kustomize }}" }} edit set namespace {{ "{{ namespace }}" }}'
      args:
        chdir: '{{ "{{ config_dir }}" }}/testing'
`
