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
	"path/filepath"
	"strings"
	"text/template"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &CR{}
var _ file.UseCustomFuncMap = &CR{}

// CR scaffolds a sample manifest for a CRD.
type CR struct {
	file.TemplateMixin
	file.ResourceMixin

	Spec string
}

// SetTemplateDefaults implements input.Template
func (f *CR) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "samples", "%[group]_%[version]_%[kind].yaml")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.IfExistsAction = file.Error

	if len(f.Spec) == 0 {
		f.Spec = defaultSpecTemplate
	}

	f.TemplateBody = crSampleTemplate
	return nil
}

func indent(spaces int, v string) string {
	pad := strings.Repeat(" ", spaces)
	return pad + strings.Replace(v, "\n", "\n"+pad, -1)
}

// GetFuncMap implements file.UseCustomFuncMap
func (f *CR) GetFuncMap() template.FuncMap {
	fm := file.DefaultFuncMap()
	fm["indent"] = indent
	return fm
}

const defaultSpecTemplate = `foo: bar`

const crSampleTemplate = `apiVersion: {{ .Resource.Domain }}/{{ .Resource.Version }}
kind: {{ .Resource.Kind }}
metadata:
  name: {{ lower .Resource.Kind }}-sample
spec:
{{ .Spec | indent 2 }}
`
