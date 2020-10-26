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

package testing

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Kustomization{}

// Kustomization scaffolds the kustomization file for use
// during Ansible testing
type Kustomization struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *Kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "testing", "kustomization.yaml")
	}

	f.TemplateBody = KustomizationTemplate

	f.IfExistsAction = file.Error

	return nil
}

const KustomizationTemplate = `# Adds namespace to all resources.
namespace: osdk-test

namePrefix: osdk-

# Labels to add to all resources and selectors.
#commonLabels:
#  someName: someValue

patchesStrategicMerge:
- manager_image.yaml
- debug_logs_patch.yaml
- ../default/manager_auth_proxy_patch.yaml

apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- ../crd
- ../rbac
- ../manager
images:
- name: testing
  newName: testing-operator
`
