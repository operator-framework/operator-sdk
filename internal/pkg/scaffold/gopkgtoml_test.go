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
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
)

func TestGopkgtoml(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &GopkgToml{})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if gopkgtomlExp != buf.String() {
		diffs := diffutil.Diff(gopkgtomlExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const gopkgtomlExp = `# Force dep to vendor the code generators, which aren't imported just used at dev time.
required = [
  "k8s.io/code-generator/cmd/defaulter-gen",
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
  # revision for tag "kubernetes-1.13.1"
  revision = "c2090bec4d9b1fb25de3812f868accc2bc9ecbae"

[[override]]
  name = "k8s.io/kube-openapi"
  revision = "0cf8f7e6ed1d2e3d47d02e3b6e559369af24d803"

[[override]]
  name = "github.com/go-openapi/spec"
  branch = "master"

[[override]]
  name = "sigs.k8s.io/controller-tools"
  version = "=v0.1.8"

[[override]]
  name = "k8s.io/api"
  # revision for tag "kubernetes-1.13.1"
  revision = "05914d821849570fba9eacfb29466f2d8d3cd229"

[[override]]
  name = "k8s.io/apiextensions-apiserver"
  # revision for tag "kubernetes-1.13.1"
  revision = "0fe22c71c47604641d9aa352c785b7912c200562"

[[override]]
  name = "k8s.io/apimachinery"
  # revision for tag "kubernetes-1.13.1"
  revision = "2b1284ed4c93a43499e781493253e2ac5959c4fd"

[[override]]
  name = "k8s.io/client-go"
  # revision for tag "kubernetes-1.13.1"
  revision = "8d9ed539ba3134352c586810e749e58df4e94e4f"

[[override]]
  name = "github.com/coreos/prometheus-operator"
  version = "=v0.29.0"

[[override]]
  name = "sigs.k8s.io/controller-runtime"
  version = "=v0.1.10"

[[constraint]]
  name = "github.com/operator-framework/operator-sdk"
  # The version rule is used for a specific release and the master branch for in between releases.
  # branch = "master" #osdk_branch_annotation
  version = "=v0.7.0" #osdk_version_annotation

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
