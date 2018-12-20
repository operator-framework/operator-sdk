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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

func TestCRDGoProject(t *testing.T) {
	r, err := NewResource("cache.example.com/v1alpha1", "Memcached")
	if err != nil {
		t.Fatal(err)
	}
	s, buf := setupScaffoldAndWriter()
	absPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Set the project and repo paths to {abs}/_testdata, which contains pkg/apis
	// for the memcached-operator.
	td := "_testdata"
	repo := absPath[strings.Index(absPath, "github.com"):]
	cfg := &input.Config{
		Repo:           filepath.Join(repo, td),
		AbsProjectPath: filepath.Join(absPath, td),
		ProjectName:    td,
	}
	if err := os.Chdir(cfg.AbsProjectPath); err != nil {
		t.Fatal(err)
	}
	defer func() { os.Chdir(absPath) }()
	err = s.Execute(cfg, &Crd{Resource: r, IsOperatorGo: true})
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if crdGoExp != buf.String() {
		diffs := diffutil.Diff(crdGoExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

const crdGoExp = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  name: memcacheds.cache.example.com
spec:
  group: cache.example.com
  names:
    kind: Memcached
    listKind: MemcachedList
    plural: memcacheds
    singular: memcached
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          properties:
            size:
              format: int32
              type: integer
          required:
          - size
          type: object
        status:
          properties:
            nodes:
              items:
                type: string
              type: array
          required:
          - nodes
          type: object
  version: v1alpha1
`

func TestCrdNonGoProject(t *testing.T) {
	r, err := NewResource(appApiVersion, appKind)
	if err != nil {
		t.Fatal(err)
	}
	s, buf := setupScaffoldAndWriter()
	err = s.Execute(appConfig, &Crd{Resource: r})
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if crdNonGoExp != buf.String() {
		diffs := diffutil.Diff(crdNonGoExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

const crdNonGoExp = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  creationTimestamp: null
  name: appservices.app.example.com
spec:
  group: app.example.com
  names:
    kind: AppService
    listKind: AppServiceList
    plural: appservices
    singular: appservice
  scope: Namespaced
  subresources:
    status: {}
  version: v1alpha1
`
