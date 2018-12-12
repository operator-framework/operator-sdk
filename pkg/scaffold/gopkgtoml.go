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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	"github.com/BurntSushi/toml"
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
  # revision for tag "kubernetes-1.12.3"
  revision = "3dcf91f64f638563e5106f21f50c31fa361c918d"

[[override]]
  name = "k8s.io/api"
  # revision for tag "kubernetes-1.12.3"
  revision = "b503174bad5991eb66f18247f52e41c3258f6348"

[[override]]
  name = "k8s.io/apiextensions-apiserver"
  # revision for tag "kubernetes-1.12.3"
  revision = "0cd23ebeb6882bd1cdc2cb15fc7b2d72e8a86a5b"

[[override]]
  name = "k8s.io/apimachinery"
  # revision for tag "kubernetes-1.12.3"
  revision = "eddba98df674a16931d2d4ba75edc3a389bf633a"

[[override]]
  name = "k8s.io/client-go"
  # revision for tag "kubernetes-1.12.3"
  revision = "d082d5923d3cc0bfbb066ee5fbdea3d0ca79acf8"

[[override]]
  name = "sigs.k8s.io/controller-runtime"
  version = "=v0.1.8"

[[constraint]]
  name = "github.com/operator-framework/operator-sdk"
  # The version rule is used for a specific release and the master branch for in between releases.
  # branch = "master" #osdk_branch_annotation
  version = "=v0.3.0" #osdk_version_annotation

[prune]
  go-tests = true
  non-go = true

  [[prune.project]]
    name = "k8s.io/code-generator"
    non-go = false
`

func PrintDepsAsFile() {
	fmt.Println(gopkgTomlTmpl)
}

func PrintDeps() error {
	gopkgData := make(map[string]interface{})
	_, err := toml.Decode(gopkgTomlTmpl, &gopkgData)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}
	w := tabwriter.NewWriter(buf, 16, 8, 0, '\t', 0)
	_, err = w.Write([]byte("NAME\tVERSION\tBRANCH\tREVISION\t\n"))
	if err != nil {
		return err
	}

	constraintList, ok := gopkgData["constraint"]
	if !ok {
		return errors.New("constraints not found")
	}
	for _, dep := range constraintList.([]map[string]interface{}) {
		err = writeDepRow(w, dep)
		if err != nil {
			return err
		}
	}
	overrideList, ok := gopkgData["override"]
	if !ok {
		return errors.New("overrides not found")
	}
	for _, dep := range overrideList.([]map[string]interface{}) {
		err = writeDepRow(w, dep)
		if err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}

	requiredList, ok := gopkgData["required"]
	if !ok {
		return errors.New("required list not found")
	}
	pl, err := json.MarshalIndent(requiredList, "", " ")
	if err != nil {
		return err
	}
	_, err = buf.Write([]byte(fmt.Sprintf("\nrequired = %v", string(pl))))
	if err != nil {
		return err
	}

	fmt.Println(buf.String())

	return nil
}

func writeDepRow(w *tabwriter.Writer, dep map[string]interface{}) error {
	name := dep["name"].(string)
	ver, col := "", 0
	if v, ok := dep["version"]; ok {
		ver, col = v.(string), 1
	} else if v, ok = dep["branch"]; ok {
		ver, col = v.(string), 2
	} else if v, ok = dep["revision"]; ok {
		ver, col = v.(string), 3
	} else {
		return fmt.Errorf("no version, revision, or branch found for %s", name)
	}

	_, err := w.Write([]byte(name + strings.Repeat("\t", col) + ver + "\t\n"))
	return err
}
