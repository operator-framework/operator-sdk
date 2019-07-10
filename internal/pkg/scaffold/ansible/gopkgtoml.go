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

package ansible

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/internal/deps"
)

// GopkgToml - the Gopkg.toml file for a hybrid operator
type GopkgToml struct {
	StaticInput
}

func (s *GopkgToml) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = scaffold.GopkgTomlFile
	}
	s.TemplateBody = gopkgTomlTmpl
	return s.Input, nil
}

const gopkgTomlTmpl = `[[constraint]]
  name = "github.com/operator-framework/operator-sdk"
  # The version rule is used for a specific release and the master branch for in between releases.
  # branch = "master" #osdk_branch_annotation
  version = "=v0.9.0" #osdk_version_annotation

[[override]]
  name = "k8s.io/api"
  version = "kubernetes-1.13.4"

[[override]]
  name = "k8s.io/apimachinery"
  version = "kubernetes-1.13.4"

[[override]]
  name = "k8s.io/client-go"
  version = "kubernetes-1.13.4"

[prune]
  go-tests = true
  unused-packages = true
`

func PrintDepGopkgTOML(asFile bool) error {
	if asFile {
		_, err := fmt.Println(gopkgTomlTmpl)
		return err
	}
	return deps.PrintDepGopkgTOML(gopkgTomlTmpl)
}
