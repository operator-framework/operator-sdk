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
  version = "=v0.31.1"

[[override]]
  name = "k8s.io/kube-state-metrics"
  version = "v1.6.0"

[[override]]
  name = "sigs.k8s.io/controller-runtime"
  version = "=v0.2.0-beta.3"

# Required when resolving controller-runtime dependencies.
[[override]]
  name = "gopkg.in/fsnotify.v1"
  source = "https://github.com/fsnotify/fsnotify.git"

[[constraint]]
  name = "github.com/operator-framework/operator-sdk"
  # The version rule is used for a specific release and the master branch for in between releases.
  branch = "master" #osdk_branch_annotation
  # version = "=v0.9.0" #osdk_version_annotation

[prune]
  go-tests = true
  non-go = true

  [[prune.project]]
    name = "k8s.io/kube-state-metrics"
    unused-packages = true

`
