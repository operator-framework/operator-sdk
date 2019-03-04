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
	"reflect"
	"strings"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	"github.com/ghodss/yaml"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
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
	// Set the project and repo paths to {abs}/test/test-framework, which
	// contains pkg/apis for the memcached-operator.
	tfDir := filepath.Join("test", "test-framework")
	pkgIdx := strings.Index(absPath, "pkg")
	cfg := &input.Config{
		Repo:           filepath.Join(absPath[strings.Index(absPath, "github.com"):pkgIdx], tfDir),
		AbsProjectPath: filepath.Join(absPath[:pkgIdx], tfDir),
		ProjectName:    tfDir,
	}
	if err := os.Chdir(cfg.AbsProjectPath); err != nil {
		t.Fatal(err)
	}
	defer func() { os.Chdir(absPath) }()
	err = s.Execute(cfg, &CRD{
		Input:        input.Input{Path: filepath.Join(tfDir, "cache_v1alpha1_memcached.yaml")},
		Resource:     r,
		IsOperatorGo: true,
	})
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
  versions:
  - name: v1alpha1
    served: true
    storage: true
`

func TestCRDNonGoProject(t *testing.T) {
	r, err := NewResource(appApiVersion, appKind)
	if err != nil {
		t.Fatal(err)
	}
	s, buf := setupScaffoldAndWriter()
	err = s.Execute(appConfig, &CRD{Resource: r})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if crdNonGoExp != buf.String() {
		diffs := diffutil.Diff(crdNonGoExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

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
  version: v1alpha1
  versions:
  - name: v1alpha1
    served: true
    storage: true
`

func marshalAndGetStrings(a, b interface{}) (string, string, error) {
	var (
		ab, bb []byte
		err    error
	)
	if ab, err = yaml.Marshal(a); err != nil {
		return "", "", err
	}
	if bb, err = yaml.Marshal(b); err != nil {
		return "", "", err
	}
	return string(ab), string(bb), nil
}

var baseCRDVal = &apiextv1beta1.CustomResourceValidation{
	OpenAPIV3Schema: &apiextv1beta1.JSONSchemaProps{
		Properties: map[string]apiextv1beta1.JSONSchemaProps{
			"apiVersion": {
				Description: `APIVersion defines the versioned schema of this representation of an object.`,
				Type:        "string",
			},
			"kind": {
				Description: `Kind is a string value representing the REST resource this object represents.`,
				Type:        "string",
			},
			"metadata": {Type: "object"},
			"spec": {
				Properties: map[string]apiextv1beta1.JSONSchemaProps{
					"size": {
						Description: `Desired state of cluster.`,
						Format:      "int32",
						Type:        "integer",
					},
				},
				Required: []string{"size"},
				Type:     "object",
			},
			"status": {
				Properties: map[string]apiextv1beta1.JSONSchemaProps{
					"nodes": {
						Description: `Define observed state of cluster.`,
						Items:       &apiextv1beta1.JSONSchemaPropsOrArray{Schema: &apiextv1beta1.JSONSchemaProps{Type: "string"}},
						Type:        "array",
					},
				},
				Required: []string{"nodes"},
				Type:     "object",
			},
		},
	},
}

func TestMergeValidationsNoChange(t *testing.T) {
	// Both should be the same.
	dstCRDVal, srcCRDVal := baseCRDVal.DeepCopy(), baseCRDVal.DeepCopy()
	mergeValidations(dstCRDVal, srcCRDVal)
	if !reflect.DeepEqual(dstCRDVal, srcCRDVal) {
		ts, bs, err := marshalAndGetStrings(dstCRDVal, srcCRDVal)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("Expected vs actual differs.\n%v", diffutil.Diff(bs, ts))
	}
}

func TestMergeValidationsUpdateDescription(t *testing.T) {
	// apiVersion description should be that in src val.
	dstCRDVal, srcCRDVal := baseCRDVal.DeepCopy(), baseCRDVal.DeepCopy()
	apiv := dstCRDVal.OpenAPIV3Schema.Properties["apiVersion"]
	apiv.Description = "foo bar"
	dstCRDVal.OpenAPIV3Schema.Properties["apiVersion"] = apiv

	mergeValidations(dstCRDVal, srcCRDVal)
	if !reflect.DeepEqual(dstCRDVal, srcCRDVal) {
		ds, ss, err := marshalAndGetStrings(dstCRDVal, srcCRDVal)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("Expected vs actual differs.\n%v", diffutil.Diff(ss, ds))
	}
}

func TestMergeValidationsUpdateMultiple(t *testing.T) {
	// specs' size description should be that in src val.
	// sizes' required should be that in dst val.
	dstCRDVal, srcCRDVal := baseCRDVal.DeepCopy(), baseCRDVal.DeepCopy()
	spec := dstCRDVal.OpenAPIV3Schema.Properties["spec"]
	specSize := spec.Properties["size"]
	oldSpecDesc := specSize.Description
	specSize.Description = "new description"
	newRequired := []string{"some prop"}
	specSize.Required = newRequired
	spec.Properties["size"] = specSize
	dstCRDVal.OpenAPIV3Schema.Properties["spec"] = spec

	mergeValidations(dstCRDVal, srcCRDVal)
	if reflect.DeepEqual(dstCRDVal, srcCRDVal) {
		t.Error("Expected vs actual should differ but do not.")
	}
	specSize = dstCRDVal.OpenAPIV3Schema.Properties["spec"].Properties["size"]
	if specSize.Description != oldSpecDesc {
		t.Errorf("Expected old description %s, got %s", oldSpecDesc, specSize.Description)
	}
	if !reflect.DeepEqual(specSize.Required, newRequired) {
		t.Errorf("Expected new required %v, got %v", newRequired, specSize.Required)
	}
}

func TestMergeValidationsAddNewDeleteOld(t *testing.T) {
	// specs' size description should be that in src val.
	// sizes' required should be that in dst val.
	dstCRDVal, srcCRDVal := baseCRDVal.DeepCopy(), baseCRDVal.DeepCopy()

	updatePropSpec := apiextv1beta1.JSONSchemaProps{
		Properties: map[string]apiextv1beta1.JSONSchemaProps{
			"shape": {
				Description: `Desired shape of cluster.`,
				Format:      "bool",
				Type:        "boolean",
			},
			"curve": {
				Title: "Some curve thing.",
			},
		},
		Required: []string{"shape"},
		Type:     "object",
	}
	newPropHammer := apiextv1beta1.JSONSchemaProps{
		Properties: map[string]apiextv1beta1.JSONSchemaProps{
			"nail": {
				Type: "object",
			},
		},
		Items: &apiextv1beta1.JSONSchemaPropsOrArray{Schema: &apiextv1beta1.JSONSchemaProps{Type: "string"}},
		Type:  "object",
	}

	srcCRDVal.OpenAPIV3Schema.Properties["spec"] = updatePropSpec
	srcCRDVal.OpenAPIV3Schema.Properties["hammer"] = newPropHammer
	delete(srcCRDVal.OpenAPIV3Schema.Properties, "status")

	mergeValidations(dstCRDVal, srcCRDVal)
	if !reflect.DeepEqual(dstCRDVal, srcCRDVal) {
		ds, ss, err := marshalAndGetStrings(dstCRDVal, srcCRDVal)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("Expected vs actual differs.\n%v", diffutil.Diff(ss, ds))
	}

	spec := dstCRDVal.OpenAPIV3Schema.Properties["spec"]
	if !reflect.DeepEqual(spec, updatePropSpec) {
		ss, us, err := marshalAndGetStrings(spec, updatePropSpec)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("Expected vs actual differs.\n%v", diffutil.Diff(us, ss))
	}

	if _, ok := dstCRDVal.OpenAPIV3Schema.Properties["status"]; ok {
		t.Error("Expected no status field but still exists")
	}

	h, ok := dstCRDVal.OpenAPIV3Schema.Properties["hammer"]
	if !ok {
		t.Error("Expected hammer field but does not exist")
	}
	if !reflect.DeepEqual(h, newPropHammer) {
		hs, us, err := marshalAndGetStrings(h, newPropHammer)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("Expected vs actual differs.\n%v", diffutil.Diff(us, hs))
	}
}
