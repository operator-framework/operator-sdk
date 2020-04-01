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

//nolint:lll
package ansible

import (
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
)

const MoleculeTestLocalConvergeFile = "converge.yml"

type MoleculeTestLocalConverge struct {
	input.Input
	Resource scaffold.Resource
}

// GetInput - gets the input
func (m *MoleculeTestLocalConverge) GetInput() (input.Input, error) {
	if m.Path == "" {
		m.Path = filepath.Join(MoleculeTestLocalDir, MoleculeTestLocalConvergeFile)
	}
	m.TemplateBody = moleculeTestLocalConvergeAnsibleTmpl
	m.Delims = AnsibleDelims

	return m.Input, nil
}

const moleculeTestLocalConvergeAnsibleTmpl = `---
- name: Build Operator in Kubernetes docker container
  hosts: k8s
  collections:
   - community.kubernetes

  vars:
    image: [[.Resource.FullGroup]]/[[.ProjectName]]:testing

  tasks:
    # using command so we don't need to install any dependencies
    - name: Get existing image hash
      command: docker images -q {{ image }}
      register: prev_hash_raw
      changed_when: false

    - name: Build Operator Image
      command: docker build -f /build/build/Dockerfile -t {{ image }} /build
      register: build_cmd
      changed_when: not hash or (hash and hash not in cmd_out)
      vars:
        hash: '{{ prev_hash_raw.stdout }}'
        cmd_out: '{{ "".join(build_cmd.stdout_lines[-2:]) }}'

- name: Converge
  hosts: localhost
  connection: local
  collections:
   - community.kubernetes

  vars:
    image: [[.Resource.FullGroup]]/[[.ProjectName]]:testing
    operator_template: "{{ '/'.join([template_dir, 'operator.yaml.j2']) }}"

  tasks:
    - name: Create the Operator Deployment
      k8s:
        namespace: '{{ namespace }}'
        definition: "{{ lookup('template', operator_template) }}"
        wait: yes
      vars:
        pull_policy: Never
`
