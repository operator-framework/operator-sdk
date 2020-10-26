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
package manifests

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &kustomization{}
var _ file.Inserter = &kustomization{}

// kustomization scaffolds or updates the kustomization.yaml in config/samples.
type kustomization struct {
	file.TemplateMixin

	// GroupVersionKind is the sample's gvk to add to this scaffold.
	GroupVersionKind config.GVK
}

// SetTemplateDefaults implements file.Template
func (f *kustomization) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "samples", "kustomization.yaml")
	}

	f.TemplateBody = fmt.Sprintf(kustomizationTemplate, file.NewMarkerFor(f.Path, samplesMarker))

	return nil
}

const (
	samplesMarker = "manifestskustomizesamples"
)

// GetMarkers implements file.Inserter
func (f *kustomization) GetMarkers() []file.Marker {
	return []file.Marker{file.NewMarkerFor(f.Path, samplesMarker)}
}

const samplesCodeFragment = `- %s
`

// makeCRFileName returns a Custom Resource example file name in the same format
// as kubebuilder's CreateAPI plugin for a gvk.
func makeCRFileName(gvk config.GVK) string {
	return fmt.Sprintf("%s_%s_%s.yaml", gvk.Group, gvk.Version, strings.ToLower(gvk.Kind))
}

// GetCodeFragments implements file.Inserter
func (f *kustomization) GetCodeFragments() file.CodeFragmentsMap {
	return file.CodeFragmentsMap{
		file.NewMarkerFor(f.Path, samplesMarker): []string{fmt.Sprintf(samplesCodeFragment, makeCRFileName(f.GroupVersionKind))},
	}
}

const kustomizationTemplate = `## Append samples you want in your CSV to this file as resources ##
resources:
%s
`
