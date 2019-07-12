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
  "sigs.k8s.io/controller-tools/pkg/crd/generator",
]

[[override]]
  name = "github.com/go-openapi/spec"
  branch = "master"

[[override]]
  name = "sigs.k8s.io/controller-tools"
  revision = "9d55346c2bde73fb3326ac22eac2e5210a730207"

[[override]]
  name = "k8s.io/api"
  # revision for tag "kubernetes-1.13.4"
  revision = "5cb15d34447165a97c76ed5a60e4e99c8a01ecfe"

[[override]]
  name = "k8s.io/apiextensions-apiserver"
  # revision for tag "kubernetes-1.13.4"
  revision = "d002e88f6236312f0289d9d1deab106751718ff0"

[[override]]
  name = "k8s.io/apimachinery"
  # revision for tag "kubernetes-1.13.4"
  revision = "86fb29eff6288413d76bd8506874fddd9fccdff0"

[[override]]
  name = "k8s.io/client-go"
  # revision for tag "kubernetes-1.13.4"
  revision = "b40b2a5939e43f7ffe0028ad67586b7ce50bb675"

[[override]]
  name = "github.com/coreos/prometheus-operator"
  version = "=v0.29.0"

[[override]]
  name = "k8s.io/kube-state-metrics"
  version = "v1.6.0"

[[override]]
  name = "sigs.k8s.io/controller-runtime"
  version = "=v0.1.12"

[[constraint]]
  name = "github.com/operator-framework/operator-sdk"
  # The version rule is used for a specific release and the master branch for in between releases.
  # branch = "master" #osdk_branch_annotation
  version = "=v0.9.0" #osdk_version_annotation

[prune]
  go-tests = true
  non-go = true

  [[prune.project]]
    name = "k8s.io/kube-state-metrics"
    unused-packages = true

`

func PrintDepGopkgTOML(asFile bool) error {
	if asFile {
		_, err := fmt.Println(gopkgTomlTmpl)
		return err
	}
	return deps.PrintDepGopkgTOML(gopkgTomlTmpl)
}
