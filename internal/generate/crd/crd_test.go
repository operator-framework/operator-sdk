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

//nolint:lll
package crd

import (
	"encoding/base32"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	gen "github.com/operator-framework/operator-sdk/internal/generate/gen"
	"github.com/operator-framework/operator-sdk/internal/scaffold"

	"github.com/stretchr/testify/assert"
)

const (
	testGroup   = "cache.example.com"
	testVersion = "v1alpha1"
	testKind    = "Memcached"
)

var (
	testDataDir    = filepath.Join("..", "testdata")
	testGoDataDir  = filepath.Join(testDataDir, "go")
	testAPIVersion = path.Join(testGroup, testVersion)
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// randomString returns a base32-encoded random string, reasonably sized for
// directory creation.
func randomString() string {
	rb := []byte(strconv.Itoa(rand.Int() % (2 << 20)))
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rb)
}

func TestGenerate(t *testing.T) {
	tmp := filepath.Join(os.TempDir(), randomString())
	if err := os.MkdirAll(tmp, 0755); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			t.Fatal(err)
		}
	}()
	r, err := scaffold.NewResource(testAPIVersion, testKind)
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		description string
		generator   gen.Generator
	}{
		{
			"Generate Go CRD",
			NewCRDGo(gen.Config{
				Inputs: map[string]string{
					APIsDirKey: filepath.Join(testGoDataDir, scaffold.ApisDir),
				},
				OutputDir: filepath.Join(tmp, randomString()),
			}),
		},
		{
			"Generate non-Go CRD",
			NewCRDNonGo(gen.Config{
				Inputs: map[string]string{
					APIsDirKey: filepath.Join(testGoDataDir, scaffold.ApisDir),
				},
				OutputDir: filepath.Join(tmp, randomString()),
			}, *r),
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			err = c.generator.Generate()
			if err != nil {
				t.Errorf("Wanted nil error, got: %v", err)
			}
		})
	}
}

func TestCRDGo(t *testing.T) {
	cfg := gen.Config{
		Inputs: map[string]string{
			APIsDirKey: filepath.Join(testGoDataDir, scaffold.ApisDir),
		},
	}
	g := NewCRDGo(cfg)
	fileMap, err := g.(crdGenerator).generateGo()
	if err != nil {
		t.Fatalf("Failed to execute CRD generator: %v", err)
	}
	r, err := scaffold.NewResource(testAPIVersion, testKind)
	if err != nil {
		t.Fatal(err)
	}
	if b, ok := fileMap[getFileNameForResource(*r)]; !ok {
		t.Errorf("Failed to generate CRD for %s", r)
	} else {
		assert.Equal(t, crdCustomExp, string(b))
	}
}

func TestCRDNonGo(t *testing.T) {
	cases := []struct {
		description      string
		apiVersion, kind string
		crdsDir          string
		expCRD           string
		wantErr          bool
	}{
		{
			"non-existent CRD with default structural schema",
			testAPIVersion, testKind, filepath.Join("not", "exist"), crdNonGoDefaultExp, false,
		},
		{
			"existing CRD with custom structural schema",
			testAPIVersion, testKind, filepath.Join(testGoDataDir, scaffold.CRDsDir), crdCustomExp, false,
		},
	}

	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			r, err := scaffold.NewResource(c.apiVersion, c.kind)
			if err != nil {
				t.Fatal(err)
			}
			cfg := gen.Config{
				Inputs: map[string]string{
					CRDsDirKey: c.crdsDir,
				},
			}
			g := NewCRDNonGo(cfg, *r)
			fileMap, err := g.(crdGenerator).generateNonGo()
			if err != nil {
				t.Fatalf("Error executing CRD generator: %v", err)
			}
			if b, ok := fileMap[getFileNameForResource(*r)]; !ok {
				t.Errorf("Failed to generate CRD for %s", r)
			} else {
				assert.Equal(t, c.expCRD, string(b))
			}
		})
	}
}

// crdNonGoDefaultExp is the default non-go CRD. Non-go projects don't have the
// luxury of kubebuilder annotations.
const crdNonGoDefaultExp = `apiVersion: apiextensions.k8s.io/v1beta1
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
      type: object
      x-kubernetes-preserve-unknown-fields: true
  versions:
  - name: v1alpha1
    served: true
    storage: true
`

// crdCustomExp is a CRD with custom validation, either created manually or
// with Go API code annotations.
const crdCustomExp = `apiVersion: apiextensions.k8s.io/v1beta1
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
      description: Memcached is the Schema for the memcacheds API
      properties:
        apiVersion:
          description: 'APIVersion defines the versioned schema of this representation
            of an object. Servers should convert recognized schemas to the latest
            internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources'
          type: string
        kind:
          description: 'Kind is a string value representing the REST resource this
            object represents. Servers may infer this from the endpoint the client
            submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds'
          type: string
        metadata:
          type: object
        spec:
          description: MemcachedSpec defines the desired state of Memcached
          properties:
            size:
              description: Size is the size of the memcached deployment
              format: int32
              type: integer
          required:
          - size
          type: object
        status:
          description: MemcachedStatus defines the observed state of Memcached
          properties:
            nodes:
              description: Nodes are the names of the memcached pods
              items:
                type: string
              type: array
          required:
          - nodes
          type: object
      type: object
  version: v1alpha1
  versions:
  - name: v1alpha1
    served: true
    storage: true
`
