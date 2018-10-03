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

type testFrameworkDockerfile struct {
	in *TestFrameworkDockerfileInput
}

func NewTestFrameworkDockerfileCodegen(in *TestFrameworkDockerfileInput) Codegen {
	return &testFrameworkDockerfile{in: in}
}

type TestFrameworkDockerfileInput struct{}

func (d *testFrameworkDockerfile) Render(w io.Writer) error {
	t := template.New("test_framework_dockerfile.go")
	t, err := t.Parse(testFrameworkDockerfileTmpl)
	if err != nil {
		return err
	}

	return t.Execute(w, d.in)
}

const testFrameworkDockerfileTmpl = `ARG BASEIMAGE
FROM ${BASEIMAGE}
ADD build/_output/bin/memcached-operator-test /usr/local/bin/memcached-operator-test
ARG NAMESPACEDMAN
ADD $NAMESPACEDMAN /namespaced.yaml
ADD build/test-framework/go-test.sh /go-test.sh
`
