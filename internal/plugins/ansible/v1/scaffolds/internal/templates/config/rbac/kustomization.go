/*
Copyright 2019 The Kubernetes Authors.
Modifications copyright 2020 The Operator-SDK Authors

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
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Kustomization{}

var rbacKustomizePath = filepath.Join("config", "rbac", "kustomization.yaml")

const patch6902Marker = "patch6902"

// Kustomization scaffolds the Kustomization file in rbac folder.
type Kustomization struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *Kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = rbacKustomizePath
	}

	f.TemplateBody = fmt.Sprintf(kustomizeTemplate,
		file.NewMarkerFor(f.Path, patch6902Marker),
	)
	f.IfExistsAction = file.Error

	return nil
}

type KustomizeUpdater struct {
	file.TemplateMixin
	file.ResourceMixin
}

func (*KustomizeUpdater) GetIfExistsAction() file.IfExistsAction {
	return file.Overwrite
}

func (*KustomizeUpdater) GetPath() string {
	return rbacKustomizePath
}

func (f *KustomizeUpdater) GetMarkers() []file.Marker {
	return []file.Marker{
		file.NewMarkerFor(rbacKustomizePath, patch6902Marker),
	}
}

func (f *KustomizeUpdater) GetCodeFragments() file.CodeFragmentsMap {
	fragments := make(file.CodeFragmentsMap, 1)

	// If resource is not being provided we are creating the file, not updating it
	if f.Resource == nil {
		return fragments
	}

	// Generate patch6902 fragments
	patches := make([]string, 0)
	patches = append(patches, f.Resource.Replacer().Replace(patch6902Fragment))

	if len(patches) != 0 {
		fragments[file.NewMarkerFor(rbacKustomizePath, patch6902Marker)] = patches
	}
	return fragments
}

const kustomizeTemplate = `resources:
  - role.yaml
  - role_binding.yaml
  - leader_election_role.yaml
  - leader_election_role_binding.yaml
  # Comment the following 4 lines if you want to disable
  # the auth proxy (https://github.com/brancz/kube-rbac-proxy)
  # which protects your /metrics endpoint.
  - auth_proxy_service.yaml
  - auth_proxy_role.yaml
  - auth_proxy_role_binding.yaml
  - auth_proxy_client_clusterrole.yaml
patchesJson6902:
%s
`
const patch6902Fragment = `  - target:
      group: rbac.authorization.k8s.io
      version: v1
      kind: ClusterRole
      name: manager-role
    path: patches/%[kind]_editor_role.yaml
`
