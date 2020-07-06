/*
Copyright 2020 The Operator-SDK Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rbac

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &CRDEditorRole{}

// CRDEditorRole scaffolds the config/rbac/<kind>_editor_role.yaml
type CRDEditorRole struct {
	file.TemplateMixin
	file.ResourceMixin
}

// SetTemplateDefaults implements input.Template
func (f *CRDEditorRole) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "rbac", "patches", "%[kind]_editor_role.yaml")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = crdRoleEditorTemplate

	return nil
}

const crdRoleEditorTemplate = `---
- op: add
  path: /rules/-
  value:
    apiGroups:
      - {{ .Resource.Domain }}
    resources:
      - {{ .Resource.Plural }}
    verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
- op: add
  path: /rules/-
  value:
    apiGroups:
      - {{ .Resource.Domain }}
    resources:
      - {{ .Resource.Plural }}/status
    verbs:
      - get
      - patch
      - update
`
