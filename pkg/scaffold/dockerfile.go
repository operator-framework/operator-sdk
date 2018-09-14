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

type dockerfile struct {
	in *DockerfileInput
}

func NewDockerfileCodegen(in *DockerfileInput) Codegen {
	return &dockerfile{in: in}
}

type DockerfileInput struct {
	// ProjectName is the name of the operator project.
	ProjectName string
}

func (d *dockerfile) Render(w io.Writer) error {
	t := template.New("dockerfile.go")
	t, err := t.Parse(dockerfileTmpl)
	if err != nil {
		return err
	}

	return t.Execute(w, d.in)
}

const dockerfileTmpl = `FROM alpine:3.6

RUN adduser -D {{.ProjectName}}
USER {{.ProjectName}}

ADD build/_output/bin/{{.ProjectName}} /usr/local/bin/{{.ProjectName}}
`
