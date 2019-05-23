// Copyright 2019 The Operator-SDK Authors
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
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
)

const ToolsFile = "tools.go"

type Tools struct {
	input.Input
}

func (s *Tools) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = ToolsFile
	}
	s.TemplateBody = toolsTmpl
	return s.Input, nil
}

const toolsTmpl = `// +build tools

package tools

import (
	// Code generators built at runtime.
	_ "k8s.io/code-generator/cmd/client-gen"
	_ "k8s.io/code-generator/cmd/conversion-gen"
	_ "k8s.io/code-generator/cmd/deepcopy-gen"
	_ "k8s.io/code-generator/cmd/informer-gen"
	_ "k8s.io/code-generator/cmd/lister-gen"
	_ "k8s.io/gengo/args"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
	_ "sigs.k8s.io/controller-tools/pkg/crd/generator"
)
`
