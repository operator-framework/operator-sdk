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

type addToScheme struct {
	addToSchemeInput *AddToSchemeInput
}

// AddToSchemeInput is the input needed to generate an addtoscheme_<group>_<kind>.go file
type AddToSchemeInput struct {
	// ProjectPath is the project path rooted at GOPATH.
	ProjectPath string
	// Resource defines the inputs for the new api
	Resource *Resource
}

func NewAddToSchemeCodegen(input *AddToSchemeInput) Codegen {
	return &addToScheme{addToSchemeInput: input}
}

func (c *addToScheme) Render(w io.Writer) error {
	t := template.New("addToScheme.go")
	t, err := t.Parse(addToSchemeTemplate)
	if err != nil {
		return err
	}

	return t.Execute(w, c.addToSchemeInput)
}

const addToSchemeTemplate = `package apis

import (
	"{{ .ProjectPath }}/pkg/apis/{{ .Resource.Group }}/{{ .Resource.Version }}"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, {{ .Resource.Version }}.SchemeBuilder.AddToScheme)
}
`
