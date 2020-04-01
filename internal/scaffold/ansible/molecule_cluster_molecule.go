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

const MoleculeClusterMoleculeFile = "molecule.yml"

type MoleculeClusterMolecule struct {
	input.Input
	Resource scaffold.Resource
}

// GetInput - gets the input
func (m *MoleculeClusterMolecule) GetInput() (input.Input, error) {
	if m.Path == "" {
		m.Path = filepath.Join(MoleculeClusterDir, MoleculeClusterMoleculeFile)
	}
	m.TemplateBody = moleculeClusterMoleculeAnsibleTmpl
	m.Delims = AnsibleDelims

	return m.Input, nil
}

const moleculeClusterMoleculeAnsibleTmpl = `---
dependency:
  name: galaxy
driver:
  name: delegated
lint: |
  set -e
  yamllint -d "{extends: relaxed, rules: {line-length: {max: 120}}}" .
platforms:
- name: cluster
  groups:
  - k8s
provisioner:
  name: ansible
  lint: |
    set -e
    ansible-lint
  inventory:
    group_vars:
      all:
        namespace: ${TEST_OPERATOR_NAMESPACE:-osdk-test}
    host_vars:
      localhost:
        ansible_python_interpreter: '{{ ansible_playbook_python }}'
        deploy_dir: ${MOLECULE_PROJECT_DIRECTORY}/deploy
        template_dir: ${MOLECULE_PROJECT_DIRECTORY}/molecule/templates
        operator_image: ${OPERATOR_IMAGE:-""}
        operator_pull_policy: ${OPERATOR_PULL_POLICY:-"Always"}
  env:
    K8S_AUTH_KUBECONFIG: ${KUBECONFIG:-"~/.kube/config"}
verifier:
  name: ansible
  lint: |
    set -e
    ansible-lint
`
