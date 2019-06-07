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
<<<<<<< HEAD
=======
  name = "k8s.io/code-generator"
  # revision for tag "kubernetes-1.14.1"
  revision = "50b561225d70b3eb79a1faafd3dfe7b1a62cbe73"

[[override]]
  name = "k8s.io/kube-openapi"
  revision = "a01b7d5d6c2258c80a4a10070f3dee9cd575d9c7"

[[override]]
>>>>>>> bump operator dep manager files to controller-runtime v0.2.0-beta.1 and all k8s.io deps to kubernetes-1.14.1
  name = "github.com/go-openapi/spec"
  branch = "master"

[[override]]
  name = "sigs.k8s.io/controller-tools"
  revision = "9d55346c2bde73fb3326ac22eac2e5210a730207"

[[override]]
  name = "k8s.io/api"
  # revision for tag "kubernetes-1.14.1"
  revision = "6e4e0e4f393bf5e8bbff570acd13217aa5a770cd"

[[override]]
  name = "k8s.io/apiextensions-apiserver"
  # revision for tag "kubernetes-1.14.1"
  revision = "727a075fdec8319bf095330e344b3ccc668abc73"

[[override]]
  name = "k8s.io/apimachinery"
  # revision for tag "kubernetes-1.14.1"
  revision = "6a84e37a896db9780c75367af8d2ed2bb944022e"

[[override]]
  name = "k8s.io/client-go"
  # revision for tag "kubernetes-1.14.1"
  revision = "1a26190bd76a9017e289958b9fba936430aa3704"

[[override]]
  name = "github.com/coreos/prometheus-operator"
  version = "=v0.29.0"

[[override]]
  name = "sigs.k8s.io/controller-runtime"
  version = "=v0.2.0-beta.1"

[[constraint]]
  name = "github.com/operator-framework/operator-sdk"
  # The version rule is used for a specific release and the master branch for in between releases.
  branch = "master" #osdk_branch_annotation
  # version = "=v0.8.0" #osdk_version_annotation

[prune]
  go-tests = true
  non-go = true
`

func PrintDepGopkgTOML(asFile bool) error {
	if asFile {
		_, err := fmt.Println(gopkgTomlTmpl)
		return err
	}
	return deps.PrintDepGopkgTOML(gopkgTomlTmpl)
}
