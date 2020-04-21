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
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/resource"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

func newTestUnstructured(containers []interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind":       "MyResource",
			"apiVersion": "myApi",
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

func newTestDeployment(containers []v1.Container) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
		Spec: appsv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: containers,
				},
			},
		},
	}
}

func TestManagerGenerateStrategicMergePatch(t *testing.T) {

	tests := []struct {
		o1        runtime.Object
		o2        runtime.Object
		patch     string
		patchType apitypes.PatchType
	}{
		{
			o1: newTestUnstructured([]interface{}{
				map[string]interface{}{
					"name": "test1",
				},
				map[string]interface{}{
					"name": "test2",
				},
			}),
			o2: newTestUnstructured([]interface{}{
				map[string]interface{}{
					"name": "test1",
				},
			}),
			patch:     ``,
			patchType: apitypes.JSONPatchType,
		},
		{
			o1: newTestUnstructured([]interface{}{
				map[string]interface{}{
					"name": "test1",
				},
			}),
			o2: newTestUnstructured([]interface{}{
				map[string]interface{}{
					"name": "test1",
				},
				map[string]interface{}{
					"name": "test2",
				},
			}),
			patch:     `[{"op":"add","path":"/spec/template/spec/containers/1","value":{"name":"test2"}}]`,
			patchType: apitypes.JSONPatchType,
		},
		{
			o1: newTestUnstructured([]interface{}{
				map[string]interface{}{
					"name": "test1",
				},
			}),
			o2: newTestUnstructured([]interface{}{
				map[string]interface{}{
					"name": "test1",
					"test": nil,
				},
			}),
			patch:     ``,
			patchType: apitypes.JSONPatchType,
		},
		{
			o1: newTestUnstructured([]interface{}{
				map[string]interface{}{
					"name": "test1",
				},
			}),
			o2: newTestUnstructured([]interface{}{
				map[string]interface{}{
					"name": "test2",
				},
			}),
			patch:     `[{"op":"replace","path":"/spec/template/spec/containers/0/name","value":"test2"}]`,
			patchType: apitypes.JSONPatchType,
		},
		{
			o1: newTestDeployment([]v1.Container{
				{Name: "test1"},
				{Name: "test2"},
			}),
			o2: newTestDeployment([]v1.Container{
				{Name: "test1"},
			}),
			patch:     `{"spec":{"template":{"spec":{"$setElementOrder/containers":[{"name":"test1"}]}}}}`, //nolint:lll
			patchType: apitypes.StrategicMergePatchType,
		},
		{
			o1: newTestDeployment([]v1.Container{
				{Name: "test1"},
			}),
			o2: newTestDeployment([]v1.Container{
				{Name: "test1"},
				{Name: "test2"},
			}),
			patch:     `{"spec":{"template":{"spec":{"$setElementOrder/containers":[{"name":"test1"},{"name":"test2"}],"containers":[{"name":"test2","resources":{}}]}}}}`, //nolint:lll
			patchType: apitypes.StrategicMergePatchType,
		},
		{
			o1: newTestDeployment([]v1.Container{
				{Name: "test1"},
			}),
			o2: newTestDeployment([]v1.Container{
				{Name: "test1", LivenessProbe: nil},
			}),
			patch:     `{}`,
			patchType: apitypes.StrategicMergePatchType,
		},
		{
			o1: newTestDeployment([]v1.Container{
				{Name: "test1"},
			}),
			o2: newTestDeployment([]v1.Container{
				{Name: "test2"},
			}),
			patch:     `{"spec":{"template":{"spec":{"$setElementOrder/containers":[{"name":"test2"}],"containers":[{"name":"test2","resources":{}}]}}}}`, //nolint:lll
			patchType: apitypes.StrategicMergePatchType,
		},
		{
			o1: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "ns",
					Annotations: map[string]string{
						"testannotation": "testvalue",
					},
				},
				Spec: appsv1.DeploymentSpec{},
			},
			o2: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "ns",
				},
				Spec: appsv1.DeploymentSpec{},
			},
			patch:     `{}`, //nolint:lll
			patchType: apitypes.StrategicMergePatchType,
		},
	}

	for _, test := range tests {

		o2Info := &resource.Info{
			Object: test.o2,
		}

		diff, patchType, err := createPatch(test.o1, o2Info)
		assert.NoError(t, err)
		assert.Equal(t, test.patchType, patchType)
		assert.Equal(t, test.patch, string(diff))
	}
}
