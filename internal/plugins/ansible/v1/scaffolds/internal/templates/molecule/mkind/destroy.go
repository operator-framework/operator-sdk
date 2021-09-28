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

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Destroy{}

// Destroy scaffolds a Destroy for building a main
type Destroy struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Destroy) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("molecule", "kind", "destroy.yml")
	}
	f.TemplateBody = destroyTemplate
	return nil
}

const destroyTemplate = `---
- name: Destroy
  hosts: localhost
  connection: local
  gather_facts: false
  collections:
    - kubernetes.core

  tasks:
    - name: Destroy test kind cluster
      command: kind delete cluster --name osdk-test --kubeconfig {{ "{{ kubeconfig }}" }}

    - name: Unset pull policy
      command: '{{ "{{ kustomize }}" }} edit remove patch pull_policy/{{ "{{ operator_pull_policy }}" }}.yaml'
      args:
        chdir: '{{ "{{ config_dir }}" }}/testing'
`
