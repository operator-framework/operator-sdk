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

	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
)

const MoleculeDefaultVerifyFile = "verify.yml"

type MoleculeDefaultVerify struct {
	StaticInput
}

// GetInput - gets the input
func (m *MoleculeDefaultVerify) GetInput() (input.Input, error) {
	if m.Path == "" {
		m.Path = filepath.Join(MoleculeDefaultDir, MoleculeDefaultVerifyFile)
	}
	m.TemplateBody = moleculeDefaultVerifyAnsibleTmpl

	return m.Input, nil
}

const moleculeDefaultVerifyAnsibleTmpl = `---
- name: Verify
  hosts: localhost
  connection: local
  tasks:
    - name: Get all pods in {{ namespace }}
      k8s_info:
        api_version: v1
        kind: Pod
        namespace: '{{ namespace }}'
      register: pods

    - name: Output pods
      debug: var=pods

    - name: Example assertion
      assert:
        that: true
`
