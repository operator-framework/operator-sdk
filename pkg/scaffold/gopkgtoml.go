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
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

const GopkgTomlFile = "Gopkg.toml"

type GopkgToml struct {
	input.Input
}

func (s *GopkgToml) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = GopkgTomlFile
	}
	s.TemplateBody = gopkgTomlTmpl
	return s.Input, nil
}

const gopkgTomlTmpl = `# Force dep to vendor the code generators, which aren't imported just used at dev time.
required = [
  "k8s.io/code-generator/cmd/defaulter-gen",
  "k8s.io/code-generator/cmd/deepcopy-gen",
  "k8s.io/code-generator/cmd/conversion-gen",
  "k8s.io/code-generator/cmd/client-gen",
  "k8s.io/code-generator/cmd/lister-gen",
  "k8s.io/code-generator/cmd/informer-gen",
  "k8s.io/code-generator/cmd/openapi-gen",
  "k8s.io/gengo/args",
]

[[override]]
  name = "k8s.io/code-generator"
  # revision for tag "kubernetes-1.11.2"
  revision = "6702109cc68eb6fe6350b83e14407c8d7309fd1a"

[[override]]
  name = "k8s.io/api"
  # revision for tag "kubernetes-1.11.2"
  revision = "2d6f90ab1293a1fb871cf149423ebb72aa7423aa"

[[override]]
  name = "k8s.io/apiextensions-apiserver"
  # revision for tag "kubernetes-1.11.2"
  revision = "408db4a50408e2149acbd657bceb2480c13cb0a4"

[[override]]
  name = "k8s.io/apimachinery"
  # revision for tag "kubernetes-1.11.2"
  revision = "103fd098999dc9c0c88536f5c9ad2e5da39373ae"

[[override]]
  name = "k8s.io/client-go"
  # revision for tag "kubernetes-1.11.2"
  revision = "1f13a808da65775f22cbf47862c4e5898d8f4ca1"

[[override]]
  name = "sigs.k8s.io/controller-runtime"
  version = "v0.1.4"

[[constraint]]
  name = "github.com/operator-framework/operator-sdk"
  # The version rule is used for a specific release and the master branch for in between releases.
  branch = "master" #osdk_branch_annotation
  # version = "=v0.1.0" #osdk_version_annotation

[prune]
  go-tests = true
  non-go = true
  
  [[prune.project]]
    name = "k8s.io/code-generator"
    non-go = false
`
