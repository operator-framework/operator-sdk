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

package release

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func newTestDeployment(containers []interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "Deployment",
			"apiVersion": "apps/v1",
			"metadata": map[string]interface{}{
				"name":      "test",
				"namespace": "ns",
			},
			"spec": map[string]interface{}{
				"template": map[string]interface{}{
					"spec": map[string]interface{}{
						"containers": containers,
					},
				},
			},
		},
	}
}

func TestManagerGeneratePatch(t *testing.T) {

	tests := []struct {
		o1    *unstructured.Unstructured
		o2    *unstructured.Unstructured
		patch []map[string]interface{}
	}{
		{
			o1: newTestDeployment([]interface{}{
				map[string]interface{}{
					"name": "test1",
				},
				map[string]interface{}{
					"name": "test2",
				},
			}),
			o2: newTestDeployment([]interface{}{
				map[string]interface{}{
					"name": "test1",
				},
			}),
			patch: []map[string]interface{}{},
		},
		{
			o1: newTestDeployment([]interface{}{
				map[string]interface{}{
					"name": "test1",
				},
			}),
			o2: newTestDeployment([]interface{}{
				map[string]interface{}{
					"name": "test1",
				},
				map[string]interface{}{
					"name": "test2",
				},
			}),
			patch: []map[string]interface{}{
				{
					"op":   "add",
					"path": "/spec/template/spec/containers/1",
					"value": map[string]interface{}{
						"name": string("test2"),
					},
				},
			},
		},
		{
			o1: newTestDeployment([]interface{}{
				map[string]interface{}{
					"name": "test1",
				},
			}),
			o2: newTestDeployment([]interface{}{
				map[string]interface{}{
					"name": "test1",
					"test": nil,
				},
			}),
			patch: []map[string]interface{}{},
		},
		{
			o1: newTestDeployment([]interface{}{
				map[string]interface{}{
					"name": "test1",
				},
			}),
			o2: newTestDeployment([]interface{}{
				map[string]interface{}{
					"name": "test2",
				},
			}),
			patch: []map[string]interface{}{
				{
					"op":    "replace",
					"path":  "/spec/template/spec/containers/0/name",
					"value": "test2",
				},
			},
		},
	}

	for _, test := range tests {
		diff, err := generatePatch(test.o1, test.o2)
		assert.NoError(t, err)

		if len(test.patch) == 0 {
			assert.Equal(t, 0, len(test.patch))
		} else {
			x := []map[string]interface{}{}
			err = json.Unmarshal(diff, &x)
			assert.NoError(t, err)
			assert.Equal(t, test.patch, x)
		}
	}
}
