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

package crd

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var (
	_ machinery.Template = &Kustomization{}
	_ machinery.Inserter = &Kustomization{}
)

// Kustomization scaffolds the kustomization file in manager folder.
type Kustomization struct {
	machinery.TemplateMixin
	machinery.ResourceMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "crd", "kustomization.yaml")
	}

	f.TemplateBody = fmt.Sprintf(kustomizationTemplate, machinery.NewMarkerFor(f.Path, resourceMarker))

	return nil
}

const (
	resourceMarker = "crdkustomizeresource"
)

// GetMarkers implements machinery.Inserter
func (f *Kustomization) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(f.Path, resourceMarker),
	}
}

const (
	resourceCodeFragment = `- bases/%s_%s.yaml
`
)

// GetCodeFragments implements machinery.Inserter
func (f *Kustomization) GetCodeFragments() machinery.CodeFragmentsMap {
	return machinery.CodeFragmentsMap{
		// Generate resource code fragments
		machinery.NewMarkerFor(f.Path, resourceMarker): []string{
			fmt.Sprintf(resourceCodeFragment, f.Resource.QualifiedGroup(), f.Resource.Plural),
		},
	}
}

var kustomizationTemplate = `# This kustomization.yaml is not intended to be run by itself,
# since it depends on service name and namespace that are out of this kustomize package.
# It should be run by config/default
resources:
%s
`
