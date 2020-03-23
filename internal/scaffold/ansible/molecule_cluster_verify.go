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

const MoleculeClusterVerifyFile = "verify.yml"

type MoleculeClusterVerify struct {
	input.Input
	Resource scaffold.Resource
}

// GetInput - gets the input
func (m *MoleculeClusterVerify) GetInput() (input.Input, error) {
	if m.Path == "" {
		m.Path = filepath.Join(MoleculeClusterDir, MoleculeClusterVerifyFile)
	}
	m.TemplateBody = moleculeClusterVerifyAnsibleTmpl
	m.Delims = AnsibleDelims

	return m.Input, nil
}

//nolint:lll
const moleculeClusterVerifyAnsibleTmpl = `---
# This is an example playbook to execute Ansible tests.
- name: Verify
  hosts: localhost
  connection: local
  gather_facts: no
  collections:
    - community.kubernetes

  vars:
    custom_resource: "{{ lookup('template', '/'.join([deploy_dir, 'crds/[[.Resource.FullGroup]]_[[.Resource.Version]]_[[.Resource.LowerKind]]_cr.yaml'])) | from_yaml }}"

  tasks:
    - name: Create the [[.Resource.FullGroup]]/[[.Resource.Version]].[[.Resource.Kind]] and wait for reconciliation to complete
      k8s:
        state: present
        namespace: '{{ namespace }}'
        definition: '{{ custom_resource }}'
        wait: yes
        wait_timeout: 300
        wait_condition:
          type: Running
          reason: Successful
          status: "True"

    - name: Get Pods
      k8s_info:
        api_version: v1
        kind: Pod
        namespace: '{{ namespace }}'
      register: pods

    - name: Example assertion
      assert:
        that: (pods | length) > 0
`
