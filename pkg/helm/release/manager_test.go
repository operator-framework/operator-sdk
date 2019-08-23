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

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
)

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
		o1    runtime.Object
		o2    runtime.Object
		patch string
	}{
		{
			o1: newTestDeployment([]v1.Container{
				{Name: "test1"},
				{Name: "test2"},
			}),
			o2: newTestDeployment([]v1.Container{
				{Name: "test1"},
			}),
			patch: `{"spec":{"template":{"spec":{"$setElementOrder/containers":[{"name":"test1"}]}}}}`, //nolint:lll
		},
		{
			o1: newTestDeployment([]v1.Container{
				{Name: "test1"},
			}),
			o2: newTestDeployment([]v1.Container{
				{Name: "test1"},
				{Name: "test2"},
			}),
			patch: `{"spec":{"template":{"spec":{"$setElementOrder/containers":[{"name":"test1"},{"name":"test2"}],"containers":[{"name":"test2","resources":{}}]}}}}`, //nolint:lll
		},
		{
			o1: newTestDeployment([]v1.Container{
				{Name: "test1"},
			}),
			o2: newTestDeployment([]v1.Container{
				{Name: "test1", LivenessProbe: nil},
			}),
			patch: `{}`,
		},
		{
			o1: newTestDeployment([]v1.Container{
				{Name: "test1"},
			}),
			o2: newTestDeployment([]v1.Container{
				{Name: "test2"},
			}),
			patch: `{"spec":{"template":{"spec":{"$setElementOrder/containers":[{"name":"test2"}],"containers":[{"name":"test2","resources":{}}]}}}}`, //nolint:lll
		},
	}

	for _, test := range tests {
		diff, err := generateStrategicMergePatch(test.o1, test.o2)
		assert.NoError(t, err)
		assert.Equal(t, test.patch, string(diff))
	}
}
