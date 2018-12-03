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

package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type mockEngine struct {
	out map[string]string
}

func (e *mockEngine) Render(chrt *chart.Chart, v chartutil.Values) (map[string]string, error) {
	return e.out, nil
}

func TestOwnerRefEngine(t *testing.T) {
	ownerRefs := []metav1.OwnerReference{
		{
			APIVersion: "v1",
			Kind:       "Test",
			Name:       "test",
			UID:        "123",
		},
	}

	baseOut := `apiVersion: stable.nicolerenee.io/v1
kind: Character
metadata:
  name: nemo
spec:
  Name: Nemo
`

	expectedOut := `---
apiVersion: stable.nicolerenee.io/v1
kind: Character
metadata:
  name: nemo
  ownerReferences:
  - apiVersion: v1
    kind: Test
    name: test
    uid: "123"
spec:
  Name: Nemo
`
	expected := map[string]string{"template.yaml": expectedOut, "template2.yaml": expectedOut}

	baseEngineOutput := map[string]string{
		"template.yaml":  baseOut,
		"template2.yaml": baseOut,
		"empty.yaml":     "",
		"comment.yaml":   "# This is empty",
	}

	engine := NewOwnerRefEngine(&mockEngine{out: baseEngineOutput}, ownerRefs)
	out, err := engine.Render(&chart.Chart{}, map[string]interface{}{})
	require.NoError(t, err)
	require.EqualValues(t, expected, out)
}

func TestOwnerRefEngine_MultiDocumentYaml(t *testing.T) {
	ownerRefs := []metav1.OwnerReference{
		{
			APIVersion: "v1",
			Kind:       "Test",
			Name:       "test",
			UID:        "123",
		},
	}

	baseOut := `kind: ConfigMap
apiVersion: v1
metadata:
  name: eighth
  data:
    name: value
---
apiVersion: v1
kind: Pod
metadata:
  name: example-test
`

	expectedOut := `---
apiVersion: v1
kind: ConfigMap
metadata:
  data:
    name: value
  name: eighth
  ownerReferences:
  - apiVersion: v1
    kind: Test
    name: test
    uid: "123"
---
apiVersion: v1
kind: Pod
metadata:
  name: example-test
  ownerReferences:
  - apiVersion: v1
    kind: Test
    name: test
    uid: "123"
`

	expected := map[string]string{"template.yaml": expectedOut}

	baseEngineOutput := map[string]string{
		"template.yaml": baseOut,
	}

	engine := NewOwnerRefEngine(&mockEngine{out: baseEngineOutput}, ownerRefs)
	out, err := engine.Render(&chart.Chart{}, map[string]interface{}{})

	require.NoError(t, err)
	require.Equal(t, expected, out)
}
