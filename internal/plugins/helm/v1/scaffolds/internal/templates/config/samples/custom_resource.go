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
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/yaml"
)

var (
	_ machinery.Template         = &CustomResource{}
	_ machinery.UseCustomFuncMap = &CustomResource{}
)

// CustomResource scaffolds a custom resource sample manifest.
type CustomResource struct {
	machinery.TemplateMixin
	machinery.ResourceMixin

	ChartPath string
	Chart     *chart.Chart
	Spec      string
}

// SetTemplateDefaults implements machinery.Template
func (f *CustomResource) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "samples", "%[group]_%[version]_%[kind].yaml")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.IfExistsAction = machinery.OverwriteFile

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

	f.TemplateBody = customResourceTemplate
	return nil
}

func indent(spaces int, v string) string {
	pad := strings.Repeat(" ", spaces)
	return pad + strings.Replace(v, "\n", "\n"+pad, -1)
}

// GetFuncMap implements machinery.UseCustomFuncMap
func (f *CustomResource) GetFuncMap() template.FuncMap {
	fm := machinery.DefaultFuncMap()
	fm["indent"] = indent
	return fm
}

const defaultSpecTemplate = `# TODO(user): Add fields here
`

const customResourceTemplate = `apiVersion: {{ .Resource.QualifiedGroup }}/{{ .Resource.Version }}
kind: {{ .Resource.Kind }}
metadata:
  name: {{ lower .Resource.Kind }}-sample
spec:
{{ .Spec | indent 2 }}
`
