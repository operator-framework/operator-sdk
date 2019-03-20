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
	"path/filepath"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

const MoleculeVerifyFile = "verify.yml"

type MoleculeVerify struct {
	input.Input
	ScenarioName string
}

// GetInput - gets the input
func (m *MoleculeVerify) GetInput() (input.Input, error) {
	if m.Path == "" {
		m.Path = filepath.Join(MoleculeDir, m.ScenarioName, MoleculeVerifyFile)
	}
	m.TemplateBody = moleculeVerifyAnsibleTmpl

	return m.Input, nil
}

const moleculeVerifyAnsibleTmpl = `---

- name: Verify
  hosts: localhost
  connection: local
  vars:
    ansible_python_interpreter: '{{"{{ ansible_playbook_python }}"}}'
  tasks:
    - name: Get all pods in {{"{{ namespace }}"}}
      k8s_facts:
        api_version: v1
        kind: Pod
        namespace: '{{"{{ namespace }}"}}'
      register: pods

    - name: Output pods
      debug: var=pods

    - name: Assert that there is at least one pod
      assert:
        that: (pods.resources | length) > 0
`
