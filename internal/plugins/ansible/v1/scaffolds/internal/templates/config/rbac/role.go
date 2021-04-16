// Copyright 2019 The Operator-SDK Authors
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

package rbac

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &ManagerRole{}

var defaultRoleFile = filepath.Join("config", "rbac", "role.yaml")

// ManagerRole scaffolds the role.yaml file
type ManagerRole struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *ManagerRole) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = defaultRoleFile
	}

	f.TemplateBody = fmt.Sprintf(roleTemplate, machinery.NewMarkerFor(f.Path, rulesMarker))

	return nil
}

var _ machinery.Inserter = &ManagerRoleUpdater{}

type ManagerRoleUpdater struct {
	machinery.ResourceMixin

	SkipDefaultRules bool
}

func (*ManagerRoleUpdater) GetPath() string {
	return defaultRoleFile
}

func (*ManagerRoleUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile
}

const (
	rulesMarker = "rules"
)

func (f *ManagerRoleUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(defaultRoleFile, rulesMarker),
	}
}

func (f *ManagerRoleUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap, 1)

	// If resource is not being provided we are creating the file, not updating it
	if f.Resource == nil {
		return fragments
	}

	buf := &bytes.Buffer{}
	tmpl := template.Must(template.New("rules").Parse(rulesFragment))
	err := tmpl.Execute(buf, f)
	if err != nil {
		panic(err)
	}

	// Generate rule fragment
	rules := []string{buf.String()}

	if len(rules) != 0 {
		fragments[machinery.NewMarkerFor(defaultRoleFile, rulesMarker)] = rules
	}
	return fragments
}

const roleTemplate = `---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: manager-role
rules:
  ##
  ## Base operator rules
  ##
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
%s
`

const rulesFragment = `  ##
  ## Rules for {{.Resource.QualifiedGroup}}/{{.Resource.Version}}, Kind: {{.Resource.Kind}}
  ##
  - apiGroups:
      - {{.Resource.QualifiedGroup}}
    resources:
      - {{.Resource.Plural}}
      - {{.Resource.Plural}}/status
      - {{.Resource.Plural}}/finalizers
    verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
`
