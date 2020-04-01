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

const MoleculeClusterConvergeFile = "converge.yml"

type MoleculeClusterConverge struct {
	StaticInput
}

// GetInput - gets the input
func (m *MoleculeClusterConverge) GetInput() (input.Input, error) {
	if m.Path == "" {
		m.Path = filepath.Join(MoleculeClusterDir, MoleculeClusterConvergeFile)
	}
	m.TemplateBody = moleculeClusterConvergeAnsibleTmpl

	return m.Input, nil
}

const moleculeClusterConvergeAnsibleTmpl = `---
- name: Converge
  hosts: localhost
  connection: local
  gather_facts: no
  collections:
    - community.kubernetes

  tasks:
    - name: Ensure operator image is set
      fail:
        msg: |
          You must specify the OPERATOR_IMAGE environment variable in order to run the
          'cluster' scenario
      when: not operator_image

    - name: Create the Operator Deployment
      k8s:
        namespace: '{{ namespace }}'
        definition: "{{ lookup('template', '/'.join([template_dir, 'operator.yaml.j2'])) }}"
        wait: yes
      vars:
        image: '{{ operator_image }}'
        pull_policy: '{{ operator_pull_policy }}'
`
