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
	"path/filepath"
	"testing"

	testutil "github.com/operator-framework/operator-sdk/internal/pkg/scaffold/internal/testutil"
	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"

	"github.com/spf13/afero"
)

func TestCRDGoProject(t *testing.T) {
	r, err := NewResource("cache.example.com/v1alpha1", "Memcached")
	if err != nil {
		t.Fatal(err)
	}
	s, buf := setupScaffoldAndWriter()
	s.Fs = afero.NewMemMapFs()
	cfg, err := setupTestFrameworkConfig()
	if err != nil {
		t.Fatal(err)
	}

	err = testutil.WriteOSPathToFS(afero.NewOsFs(), s.Fs, cfg.AbsProjectPath)
	if err != nil {
		t.Fatal(err)
	}

	err = s.Execute(cfg, &CRD{Resource: r, IsOperatorGo: true})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if crdGoExp != buf.String() {
		diffs := diffutil.Diff(crdGoExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const crdGoExp = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
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
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          properties:
            size:
              description: Size is the size of the memcached deployment
              format: int32
              type: integer
          required:
          - size
          type: object
        status:
          properties:
            nodes:
              description: Nodes are the names of the memcached pods
              items:
                type: string
              type: array
          required:
          - nodes
          type: object
  version: v1alpha1
  versions:
  - name: v1alpha1
    served: true
    storage: true
`

func TestCRDNonGoProject(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	s.Fs = afero.NewMemMapFs()

	r, err := NewResource(appApiVersion, appKind)
	if err != nil {
		t.Fatal(err)
	}

	crd := &CRD{Resource: r}
	i, err := crd.GetInput()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := setupTestFrameworkConfig()
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(cfg.AbsProjectPath, i.Path)
	err = afero.WriteFile(s.Fs, path, []byte(crdNonGoExp), fileutil.DefaultFileMode)
	if err != nil {
		t.Fatal(err)
	}

	if err = s.Execute(cfg, crd); err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if crdNonGoExp != buf.String() {
		diffs := diffutil.Diff(crdNonGoExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

// crdNonGoExp contains a simple validation block to make sure manually-added
// validation is not overwritten. Non-go projects don't have the luxury of
// kubebuilder annotations.
const crdNonGoExp = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
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
  validation:
    openAPIV3Schema:
      properties:
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
  versions:
  - name: v1alpha1
    served: true
    storage: true
`
