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

package k8sutil

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"gotest.tools/assert"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGetOperatorName(t *testing.T) {
	type Output struct {
		operatorName string
		err          error
	}

	type Scenario struct {
		name           string
		envVarKey      string
		envVarValue    string
		expectedOutput Output
	}

	tests := []Scenario{
		{
			name:        "Simple case",
			envVarKey:   OperatorNameEnvVar,
			envVarValue: "myoperator",
			expectedOutput: Output{
				operatorName: "myoperator",
				err:          nil,
			},
		},
		{
			name:        "Unset env var",
			envVarKey:   "",
			envVarValue: "",
			expectedOutput: Output{
				operatorName: "",
				err:          fmt.Errorf("%s must be set", OperatorNameEnvVar),
			},
		},
		{
			name:        "Empty env var",
			envVarKey:   OperatorNameEnvVar,
			envVarValue: "",
			expectedOutput: Output{
				operatorName: "",
				err:          fmt.Errorf("%s must not be empty", OperatorNameEnvVar),
			},
		},
	}

	for _, test := range tests {
		_ = os.Setenv(test.envVarKey, test.envVarValue)
		operatorName, err := GetOperatorName()
		if !(operatorName == test.expectedOutput.operatorName && reflect.DeepEqual(err, test.expectedOutput.err)) {
			t.Errorf("Test %s failed, expected output: %s,%v; got: %s,%v", test.name,
				test.expectedOutput.operatorName, test.expectedOutput.err, operatorName, err)
		}
		_ = os.Unsetenv(test.envVarKey)
	}
}

func TestSupportsOwnerReference(t *testing.T) {
	type testcase struct {
		name       string
		restMapper meta.RESTMapper
		owner      runtime.Object
		dependent  runtime.Object
		result     bool
	}

	var defaultVersion []schema.GroupVersion
	restMapper := meta.NewDefaultRESTMapper(defaultVersion)

	ownerGVK := schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}
	depGVK := schema.GroupVersionKind{
		Group:   "rbac.authorization.k8s.io",
		Version: "v1",
		Kind:    "ClusterRole",
	}

	restMapper.Add(ownerGVK, meta.RESTScopeNamespace)
	restMapper.Add(depGVK, meta.RESTScopeRoot)

	cases := []testcase{
		{
			name:       "This test should pass",
			restMapper: restMapper,
			owner: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "Deployment",
					"apiVersion": "apps/v1",
					"metadata": map[string]interface{}{
						"name":      "example-nginx-controller",
						"namespace": "ns",
					},
				},
			},
			dependent: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "ClusterRole",
					"apiVersion": "rbac.authorization.k8s.io/v1",
					"metadata": map[string]interface{}{
						"name":      "example-nginx-role",
						"namespace": "ns",
					},
				},
			},
			result: false,
		},
	}

	for _, c := range cases {
		useOwner, err := SupportsOwnerReference(c.restMapper, c.owner, c.dependent)
		if err != nil {
			t.Errorf("Error is %s", err)
		}
		assert.Equal(t, c.result, useOwner)
	}
}

func newTestUnstructured(containers []interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "MyKind",
			"apiVersion": "example.com/v1alpha1",
			"metadata": map[string]interface{}{
				"name":      "example-MyKind",
				"namespace": "ns",
			},
		},
	}
}
