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

type goTestScript struct {
	in *GoTestScriptInput
}

func NewGoTestScriptCodegen(in *GoTestScriptInput) Codegen {
	return &goTestScript{in: in}
}

type GoTestScriptInput struct {
	// ProjectName is the name of the operator project.
	ProjectName string
}

func (d *goTestScript) Render(w io.Writer) error {
	t := template.New("go_test_script.go")
	t, err := t.Parse(goTestScriptTmpl)
	if err != nil {
		return err
	}

	return t.Execute(w, d.in)
}

const goTestScriptTmpl = `#!/bin/sh

{{.ProjectName}}-test -test.parallel=1 -test.failfast -root=/ -kubeconfig=incluster -namespacedMan=namespaced.yaml -test.v
`
