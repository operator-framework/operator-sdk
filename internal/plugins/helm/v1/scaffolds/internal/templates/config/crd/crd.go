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

package crd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/kr/text"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &CRD{}

// CRD scaffolds a manifest for CRD sample.
type CRD struct {
	file.TemplateMixin
	file.ResourceMixin

	CRDVersion string
}

// SetTemplateDefaults implements input.Template
func (f *CRD) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "crd", "bases", fmt.Sprintf("%s_%%[plural].yaml", f.Resource.Domain))
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.IfExistsAction = file.Error

	if f.CRDVersion == "" {
		f.CRDVersion = "v1"
	} else if f.CRDVersion != "v1" && f.CRDVersion != "v1beta1" {
		return errors.New("the CRD version value must be either 'v1' or 'v1beta1'")
	}
	f.TemplateBody = fmt.Sprintf(crdTemplate,
		text.Indent(openAPIV3SchemaTemplate, "    "),
		text.Indent(openAPIV3SchemaTemplate, "      "),
	)
	return nil
}

const crdTemplate = `---
apiVersion: apiextensions.k8s.io/{{ .CRDVersion }}
kind: CustomResourceDefinition
metadata:
  name: {{ .Resource.Plural }}.{{ .Resource.Domain }}
spec:
  group: {{ .Resource.Domain }}
  names:
    kind: {{ .Resource.Kind }}
    listKind: {{ .Resource.Kind }}List
    plural: {{ .Resource.Plural }}
    singular: {{ .Resource.Kind | lower }}
  scope: Namespaced
{{- if eq .CRDVersion "v1beta1" }}
  subresources:
    status: {}
  validation:
%s
{{- end }}
  versions:
  - name: {{ .Resource.Version }}
{{- if eq .CRDVersion "v1" }}
    schema:
%s
{{- end }}
    served: true
    storage: true
{{- if eq .CRDVersion "v1" }}
    subresources:
      status: {}
{{- end }}
`

const openAPIV3SchemaTemplate = `openAPIV3Schema:
  description: {{ .Resource.Kind }} is the Schema for the {{ .Resource.Plural }} API
  properties:
    apiVersion:
      description: 'APIVersion defines the versioned schema of this representation
        of an object. Servers should convert recognized schemas to the latest
        internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
      type: string
    kind:
      description: 'Kind is a string value representing the REST resource this
        object represents. Servers may infer this from the endpoint the client
        submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
      type: string
    metadata:
      type: object
    spec:
      description: Spec defines the desired state of {{ .Resource.Kind }}
      type: object
      x-kubernetes-preserve-unknown-fields: true
    status:
      description: Status defines the observed state of {{ .Resource.Kind }}
      type: object
      x-kubernetes-preserve-unknown-fields: true
  type: object
`
