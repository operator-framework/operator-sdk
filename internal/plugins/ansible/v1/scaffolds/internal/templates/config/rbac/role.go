// Copyright 2019 The Operator-SDK Authors
// Modifications copyright 2020 The Operator-SDK Authors
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

package rbac

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Role{}

// Role scaffolds the config/rbac/auth_proxy_role.yaml file
type Role struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *Role) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "rbac", "role.yaml")
	}

	f.TemplateBody = roleTemplate

	return nil
}

const roleTemplate = `---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: manager-role
rules:
- apiGroups:
  - ""
  resources:
  - secrets
  - pods
  - pods/exec
  - pods/log
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - apps
  resources:
  - deployments
  - daemonsets
  - replicasets
  - statefulsets
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
`
