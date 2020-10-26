/*
Copyright 2018 The Kubernetes Authors.
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

package samples

import (
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
	"sigs.k8s.io/yaml"
)

var _ file.Template = &CRDSample{}
var _ file.UseCustomFuncMap = &CRDSample{}

// CRDSample scaffolds a manifest for CRD sample.
type CRDSample struct {
	file.TemplateMixin
	file.ResourceMixin

	ChartPath string
	Chart     *chart.Chart
	Spec      string
}

// SetTemplateDefaults implements input.Template
func (f *CRDSample) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "samples", "%[group]_%[version]_%[kind].yaml")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.IfExistsAction = file.Error

	if len(f.Spec) == 0 {
		f.Spec = defaultSpecTemplate
		if f.Chart != nil {
			spec, err := yaml.Marshal(f.Chart.Values)
			if err != nil {
				return fmt.Errorf("failed to get chart values: %v", err)
			}
			comment := ""
			if len(f.ChartPath) != 0 {
				comment = fmt.Sprintf("# Default values copied from <project_dir>/%s/values.yaml\n", f.ChartPath)
			}
			f.Spec = fmt.Sprintf("%s%s\n", comment, string(spec))
		}
	}

	f.TemplateBody = crdSampleTemplate
	return nil
}

func indent(spaces int, v string) string {
	pad := strings.Repeat(" ", spaces)
	return pad + strings.Replace(v, "\n", "\n"+pad, -1)
}

// GetFuncMap implements file.UseCustomFuncMap
func (f *CRDSample) GetFuncMap() template.FuncMap {
	fm := file.DefaultFuncMap()
	fm["indent"] = indent
	return fm
}

const defaultSpecTemplate = `foo: bar
`

const crdSampleTemplate = `apiVersion: {{ .Resource.Domain }}/{{ .Resource.Version }}
kind: {{ .Resource.Kind }}
metadata:
  name: {{ lower .Resource.Kind }}-sample
spec:
{{ .Spec | indent 2 }}
`
