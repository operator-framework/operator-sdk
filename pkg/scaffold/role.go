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

type role struct {
	in *RoleInput
}

// roleInput is the input needed to generate a pkg/deploy/role.yaml.
type RoleInput struct {
	// ProjectName is the name of the operator project.
	ProjectName string
}

func NewRoleCodegen(in *RoleInput) Codegen {
	return &role{in: in}
}

func (r *role) Render(w io.Writer) error {
	t := template.New("roles.go")
	t, err := t.Parse(roleTemplate)
	if err != nil {
		return err
	}

	return t.Execute(w, r.in)
}

const roleTemplate = `kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: {{.ProjectName}}
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - services
  - endpoints
  - persistentvolumeclaims
  - events
  - configmaps
  - secrets
  verbs:
  - "*"
- apiGroups:
  - apps
  resources:
  - deployments
  - daemonsets
  - replicasets
  - statefulsets
  verbs:
  - "*"
`
