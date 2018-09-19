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

type cr struct {
	crInput *CrInput
}

// CrInput is the input needed to generate a deploy/crds/<group>_<version>_<kind>_cr.yaml file
type CrInput struct {
	// Resource defines the inputs for the new api
	Resource *Resource
}

func NewCrCodegen(input *CrInput) Codegen {
	return &cr{crInput: input}
}

func (c *cr) Render(w io.Writer) error {
	t := template.New("<group>_<version>_<kind>_cr.yaml")
	t, err := t.Parse(crTemplate)
	if err != nil {
		return err
	}

	return t.Execute(w, c.crInput)
}

const crTemplate = `apiVersion: {{ .Resource.APIVersion }}
kind: {{ .Resource.Kind }}
metadata:
  name: example-{{ .Resource.LowerKind }}
spec:
  # Add fields here
  size: 3
`
