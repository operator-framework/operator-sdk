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
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

func TestCRD(t *testing.T) {
	r, err := NewResource("cache.example.com/v1alpha1", "Memcached")
	if err != nil {
		t.Fatal(err)
	}
	s, buf := setupScaffoldAndWriter()
	absPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Set the project and repo paths to {abs}/test/test-framework, which
	// contains pkg/apis for the memcached-operator.
	tfDir := filepath.Join("test", "test-framework")
	cfg := &input.Config{
		Repo:           filepath.Join("github.com", "operator-framework", "operator-sdk", tfDir),
		AbsProjectPath: filepath.Join(filepath.Dir(filepath.Dir(absPath)), tfDir),
		ProjectName:    filepath.Base(tfDir),
	}
	err = s.Execute(cfg, &Crd{Resource: r})
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if crdExp != buf.String() {
		diffs := diffutil.Diff(crdExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

const crdExp = `apiVersion: apiextensions.k8s.io/v1beta1
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
status:
  acceptedNames:
    kind: ""
    plural: ""
  conditions: null
  storedVersions: null
`
