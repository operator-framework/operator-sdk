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

package roles

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"

	"github.com/operator-framework/operator-sdk/internal/plugins/ansible/v1/constants"
)

const tasksMainPath = "tasks" + constants.FilePathSep + "main.yml"

type TasksMain struct {
	file.TemplateMixin
	file.ResourceMixin
}

// SetTemplateDefaults implements input.Template
func (f *TasksMain) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join(constants.RolesDir, "%[kind]", tasksMainPath)
		f.Path = f.Resource.Replacer().Replace(f.Path)
	}

	f.TemplateBody = tasksMainAnsibleTmpl
	return nil
}

const tasksMainAnsibleTmpl = `---
# tasks file for {{ .Resource.Kind}}
`
