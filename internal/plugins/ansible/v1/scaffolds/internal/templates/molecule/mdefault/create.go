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

var _ machinery.Template = &Create{}

// Create scaffolds a Create for building a main
type Create struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Create) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("molecule", "default", "create.yml")
	}
	f.TemplateBody = createTemplate
	return nil
}

const createTemplate = `---
- name: Create
  hosts: localhost
  connection: local
  gather_facts: false
  tasks: []
`
