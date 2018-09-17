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

type doc struct {
	docInput *DocInput
}

// DocInput is the input needed to generate a pkg/apis/<group>/<version>/doc.go file
type DocInput struct {
	// ProjectPath is the project path rooted at GOPATH.
	ProjectPath string
	// Resource defines the inputs for the new api
	Resource *Resource
}

func NewDocCodegen(input *DocInput) Codegen {
	return &doc{docInput: input}
}

func (c *doc) Render(w io.Writer) error {
	t := template.New("doc.go")
	t, err := t.Parse(docTemplate)
	if err != nil {
		return err
	}

	return t.Execute(w, c.docInput)
}

const docTemplate = `// Package {{.Resource.Version}} contains API Schema definitions for the {{ .Resource.Group }} {{.Resource.Version}} API group
// +k8s:deepcopy-gen=package,register
// +groupName={{ .Resource.FullGroup }}
package {{.Resource.Version}}
`
