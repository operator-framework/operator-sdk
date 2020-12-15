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
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v2/pkg/model/file"
)

var _ file.Template = &Verify{}

var (
	defaultVerifyFile = filepath.Join("molecule", "default", "verify.yaml")
	verifyMarker      = "failed_task_name"
)

// Verify scaffolds a Verify for building a main
type Verify struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *Verify) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = defaultVerifyFile
	}

	f.TemplateBody = fmt.Sprintf(verifyTemplate,
		file.NewMarkerFor(defaultVerifyFile, verifyMarker),
	)
	return nil
}

var _ file.Inserter = &VerifyUpdater{}

type VerifyUpdater struct {
	file.TemplateMixin
	file.ResourceMixin

	// WireResource defines the api resources are generated or not.
	WireResource bool
}

func (f *VerifyUpdater) GetPath() string {
	return defaultVerifyFile
}

func (f *VerifyUpdater) GetIfExistsAction() file.IfExistsAction {
	return file.Overwrite
}

func (f *VerifyUpdater) GetMarkers() []file.Marker {
	return []file.Marker{
		file.NewMarkerFor(defaultVerifyFile, verifyMarker),
	}
}

var taskNameCodeFragment = `%s`
var ansibleTask = `name: '{{ ansible_failed_task.name }}'`

func (f *VerifyUpdater) GetCodeFragments() file.CodeFragmentsMap {
	fragments := make(file.CodeFragmentsMap, 1)

	// If resource is not being provided we are creating the file, not updating it
	if f.Resource == nil {
		return fragments
	}

	// Generate import code fragments
	imports := make([]string, 0)
	if f.WireResource {
		fragments[file.NewMarkerFor(f.Path, verifyMarker)] = imports
	}

	// Only store code fragments in the map if the slices are non-empty
	if len(imports) != 0 {
		fragments[file.NewMarkerFor(defaultVerifyFile, verifyMarker)] = imports
	}

	return fragments
}

const verifyTemplate = `---
- name: Verify
  hosts: localhost
  connection: local
  gather_facts: no
  collections:
    - community.kubernetes

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
			  %s	
              result: '{{ "{{ ansible_failed_result }}" }}'
          fail:
            msg: '{{ "{{ failed_task }}" }}'
`
