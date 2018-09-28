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

package scaffold

import (
	"io"
	"text/template"
)

type roleBinding struct {
	in *RoleBindingInput
}

// RoleBindingInput is the input needed to generate a pkg/deploy/role_binding.yaml.
type RoleBindingInput struct {
	// ProjectName is the name of the operator project.
	ProjectName string
}

func NewRoleBindingCodegen(in *RoleBindingInput) Codegen {
	return &roleBinding{in: in}
}

func (r *roleBinding) Render(w io.Writer) error {
	t := template.New("rolebinding.go")
	t, err := t.Parse(roleBindingTemplate)
	if err != nil {
		return err
	}

	return t.Execute(w, r.in)
}

const roleBindingTemplate = `kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: default-account-{{.ProjectName}}
subjects:
- kind: ServiceAccount
  name: default
roleRef:
  kind: Role
  name: {{.ProjectName}}
  apiGroup: rbac.authorization.k8s.io
`
