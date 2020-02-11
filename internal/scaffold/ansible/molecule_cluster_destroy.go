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

package ansible

import (
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
)

const MoleculeClusterDestroyFile = "destroy.yml"

type MoleculeClusterDestroy struct {
	input.Input
	Resource scaffold.Resource
}

// GetInput - gets the input
func (m *MoleculeClusterDestroy) GetInput() (input.Input, error) {
	if m.Path == "" {
		m.Path = filepath.Join(MoleculeClusterDir, MoleculeClusterDestroyFile)
	}
	m.TemplateBody = moleculeClusterDestroyAnsibleTmpl
	m.Delims = AnsibleDelims

	return m.Input, nil
}

//nolint:lll
const moleculeClusterDestroyAnsibleTmpl = `---
- name: Destroy
  hosts: localhost
  connection: local
  gather_facts: false
  no_log: "{{ molecule_no_log }}"

  tasks:
    - name: Delete namespace
      k8s:
        api_version: v1
        kind: Namespace
        name: '{{ namespace }}'
        state: absent
        wait: yes

    - name: Delete RBAC resources
      k8s:
        definition: "{{ lookup('template', '/'.join([deploy_dir, item])) }}"
        namespace: '{{ namespace }}'
        state: absent
        wait: yes
      with_items:
        - role.yaml
        - role_binding.yaml
        - service_account.yaml

    - name: Delete Custom Resource Definition
      k8s:
        definition: "{{ lookup('file', '/'.join([deploy_dir, 'crds/[[.Resource.FullGroup]]_[[.Resource.Resource]]_crd.yaml'])) }}"
        state: absent
        wait: yes
`
