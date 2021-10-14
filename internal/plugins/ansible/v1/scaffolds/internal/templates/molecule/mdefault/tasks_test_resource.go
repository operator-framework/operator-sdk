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

var _ machinery.Template = &ResourceTest{}

// ResourceTest scaffolds a ResourceTest for building a main
type ResourceTest struct {
	machinery.TemplateMixin
	machinery.ResourceMixin
	SampleFile string
}

// SetTemplateDefaults implements machinery.Template
func (f *ResourceTest) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("molecule", "default", "tasks", "%[kind]_test.yml")
		f.Path = f.Resource.Replacer().Replace(f.Path)
	}
	f.SampleFile = f.Resource.Replacer().Replace("%[group]_%[version]_%[kind].yaml")

	f.TemplateBody = resourceTestTemplate
	return nil
}

const resourceTestTemplate = `---
- name: Create the {{ .Resource.QualifiedGroup }}/{{ .Resource.Version }}.{{ .Resource.Kind }}
  k8s:
    state: present
    namespace: '{{ "{{ namespace }}" }}'
    definition: "{{ "{{ lookup('template', '/'.join([samples_dir, cr_file])) | from_yaml }}" }}"
    wait: yes
    wait_timeout: 300
    wait_condition:
      type: Successful
      status: "True"
  vars:
    cr_file: '{{ .SampleFile }}'

- name: Add assertions here
  assert:
    that: false
    fail_msg: FIXME Add real assertions for your operator
`
