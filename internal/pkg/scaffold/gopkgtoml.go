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
	"fmt"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/internal/deps"
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
  "k8s.io/code-generator/cmd/deepcopy-gen",
  "k8s.io/code-generator/cmd/conversion-gen",
  "k8s.io/code-generator/cmd/client-gen",
  "k8s.io/code-generator/cmd/lister-gen",
  "k8s.io/code-generator/cmd/informer-gen",
  "k8s.io/kube-openapi/cmd/openapi-gen",
  "k8s.io/gengo/args",
  "sigs.k8s.io/controller-tools/pkg/crd/generator",
]

[[override]]
  name = "k8s.io/code-generator"
  version = "kubernetes-1.13.4"

[[override]]
  name = "k8s.io/kube-openapi"
  revision = "0cf8f7e6ed1d2e3d47d02e3b6e559369af24d803"

[[override]]
  name = "github.com/go-openapi/spec"
  branch = "master"

[[override]]
  name = "sigs.k8s.io/controller-tools"
  version = "kubernetes-1.13.4"

[[override]]
  name = "k8s.io/api"
  version = "kubernetes-1.13.4"

[[override]]
  name = "k8s.io/apiextensions-apiserver"
  version = "kubernetes-1.13.4"

[[override]]
  name = "k8s.io/apimachinery"
  version = "kubernetes-1.13.4"

[[override]]
  name = "k8s.io/client-go"
  version = "kubernetes-1.13.4"

[[override]]
  name = "github.com/coreos/prometheus-operator"
  version = "=v0.29.0"

[[override]]
  name = "sigs.k8s.io/controller-runtime"
  version = "=v0.2.0-alpha.0"

[[constraint]]
  name = "github.com/operator-framework/operator-sdk"
  # The version rule is used for a specific release and the master branch for in between releases.
  source = "github.com/corinnekrych/operator-sdk"
  branch = "controller-runtime-0.2.0-alpha0-vendor" #osdk_branch_annotation
  # version = "=v0.7.0" #osdk_version_annotation

[prune]
  go-tests = true
  non-go = true

  [[prune.project]]
    name = "k8s.io/code-generator"
    non-go = false

  [[prune.project]]
    name = "k8s.io/gengo"
    non-go = false
`

func PrintDepGopkgTOML(asFile bool) error {
	if asFile {
		_, err := fmt.Println(gopkgTomlTmpl)
		return err
	}
	return deps.PrintDepGopkgTOML(gopkgTomlTmpl)
}
