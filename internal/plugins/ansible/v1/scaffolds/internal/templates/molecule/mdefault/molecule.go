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

var _ file.Template = &Molecule{}

// Molecule scaffolds a Molecule for building a main
type Molecule struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *Molecule) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("molecule", "default", "molecule.yml")
	}
	f.TemplateBody = moleculeTemplate
	return nil
}

const moleculeTemplate = `---
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
        ansible_python_interpreter: '{{ "{{ ansible_playbook_python }}" }}'
        config_dir: ${MOLECULE_PROJECT_DIRECTORY}/config
        samples_dir: ${MOLECULE_PROJECT_DIRECTORY}/config/samples
        operator_image: ${OPERATOR_IMAGE:-""}
        operator_pull_policy: ${OPERATOR_PULL_POLICY:-"Always"}
        kustomize: ${KUSTOMIZE_PATH:-kustomize}
  env:
    K8S_AUTH_KUBECONFIG: ${KUBECONFIG:-"~/.kube/config"}
verifier:
  name: ansible
  lint: |
    set -e
    ansible-lint
`
