// Copyright 2018 The Operator-SDK Authors
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

package scaffold

import (
	"io"
	"text/template"
)

type register struct {
	registerInput *RegisterInput
}

// RegisterInput is the input needed to generate a pkg/apis/<group>/<version>/register.go file
type RegisterInput struct {
	// ProjectPath is the project path rooted at GOPATH.
	ProjectPath string
	// Resource defines the inputs for the new api
	Resource *Resource
}

func NewRegisterCodegen(input *RegisterInput) Codegen {
	return &register{registerInput: input}
}

func (c *register) Render(w io.Writer) error {
	t := template.New("register.go")
	t, err := t.Parse(registerTemplate)
	if err != nil {
		return err
	}

	return t.Execute(w, c.registerInput)
}

const registerTemplate = `
// NOTE: Boilerplate only.  Ignore this file.

// Package {{.Resource.Version}} contains API Schema definitions for the {{ .Resource.Group }} {{.Resource.Version}} API group
// +k8s:deepcopy-gen=package,register
// +groupName={{ .Resource.FullGroup }}
package {{.Resource.Version}}

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/runtime/scheme"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: "{{ .Resource.FullGroup }}", Version: "{{ .Resource.Version }}"}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)
`
