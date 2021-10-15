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

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Verify{}

// Verify scaffolds a Verify for building a main
type Verify struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Verify) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("molecule", "default", "verify.yml")
	}
	f.TemplateBody = verifyTemplate
	return nil
}

const verifyTemplate = `---
- name: Verify
  hosts: localhost
  connection: local
  gather_facts: no
  collections:
    - kubernetes.core

  vars:
    ctrl_label: control-plane=controller-manager

  tasks:
    - block:
        - name: Import all test files from tasks/
          include_tasks: '{{ "{{ item }}" }}'
          with_fileglob:
            - tasks/*_test.yml
      rescue:
        - name: Retrieve relevant resources
          k8s_info:
            api_version: '{{ "{{ item.api_version }}" }}'
            kind: '{{ "{{ item.kind }}" }}'
            namespace: '{{ "{{ namespace }}" }}'
          loop:
            - api_version: v1
              kind: Pod
            - api_version: apps/v1
              kind: Deployment
            - api_version: v1
              kind: Secret
            - api_version: v1
              kind: ConfigMap
          register: debug_resources

        - name: Retrieve Pod logs
          k8s_log:
            name: '{{ "{{ item.metadata.name }}" }}'
            namespace: '{{ "{{ namespace }}" }}'
            container: manager
          loop: "{{ "{{ q('k8s', api_version='v1', kind='Pod', namespace=namespace, label_selector=ctrl_label) }}" }}"
          register: debug_logs

        - name: Output gathered resources
          debug:
            var: debug_resources

        - name: Output gathered logs
          debug:
            var: item.log_lines
          loop: '{{ "{{ debug_logs.results }}" }}'

        - name: Re-emit failure
          vars:
            failed_task:
              result: '{{ "{{ ansible_failed_result }}" }}'
          fail:
            msg: '{{ "{{ failed_task }}" }}'
`
