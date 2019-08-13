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
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type mockEngine struct {
	out map[string]string
}

func (e *mockEngine) Render(chrt *chart.Chart, v chartutil.Values) (map[string]string, error) {
	return e.out, nil
}

type testCase struct {
	input    map[string]string
	expected map[string]string
}

type restMapping struct {
	gvk   schema.GroupVersionKind
	scope meta.RESTScope
}

func genTemplate(resourceCount int, withLeadingSep bool, gvk schema.GroupVersionKind, ownerRefs []metav1.OwnerReference) string {
	sb := &strings.Builder{}

	for i := 0; i < resourceCount; i++ {
		if withLeadingSep || i > 0 {
			sb.WriteString("---\n")
		}
		sb.WriteString(fmt.Sprintf("apiVersion: %s/%s\nkind: %s\nmetadata:\n  name: example-%s-%d\n", gvk.Group, gvk.Version, gvk.Kind, strings.ToLower(gvk.Kind), i))
		if len(ownerRefs) > 0 {
			sb.WriteString("  ownerReferences:\n")
		}
		for _, or := range ownerRefs {
			sb.WriteString(fmt.Sprintf("  - apiVersion: %s\n    kind: %s\n    name: %s\n    uid: %q\n", or.APIVersion, or.Kind, or.Name, or.UID))
		}
		sb.WriteString(fmt.Sprintf("spec:\n  value: value-%d\n", i))
	}
	return sb.String()
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

	ns := restMapping{
		gvk: schema.GroupVersionKind{
			Group:   "app.example.com",
			Version: "v1",
			Kind:    "App",
		},
		scope: meta.RESTScopeNamespace,
	}
	cs := restMapping{
		gvk: schema.GroupVersionKind{
			Group:   "app.example.com",
			Version: "v1",
			Kind:    "ClusterApp",
		},
		scope: meta.RESTScopeRoot,
	}
	restMapper := meta.NewDefaultRESTMapper(nil)
	restMapper.Add(ns.gvk, ns.scope)
	restMapper.Add(cs.gvk, cs.scope)

	testCases := []testCase{
		{
			input: map[string]string{
				"template1.yaml": genTemplate(1, false, ns.gvk, nil),
				"template2.yaml": genTemplate(1, false, cs.gvk, nil),
				"template3.yaml": genTemplate(2, true, ns.gvk, nil),
				"template4.yaml": genTemplate(2, true, cs.gvk, nil),
				"template5.yaml": fmt.Sprintf("%s%s",
					genTemplate(1, true, ns.gvk, nil),
					genTemplate(1, true, cs.gvk, nil),
				),
				"template6.yaml": fmt.Sprintf("%s%s",
					genTemplate(1, true, cs.gvk, nil),
					genTemplate(1, true, ns.gvk, nil),
				),
				"empty.yaml":   "",
				"comment.yaml": "# This is empty",
			},
			expected: map[string]string{
				"template1.yaml": genTemplate(1, true, ns.gvk, ownerRefs),
				"template2.yaml": genTemplate(1, true, cs.gvk, nil),
				"template3.yaml": genTemplate(2, true, ns.gvk, ownerRefs),
				"template4.yaml": genTemplate(2, true, cs.gvk, nil),
				"template5.yaml": fmt.Sprintf("%s%s",
					genTemplate(1, true, ns.gvk, ownerRefs),
					genTemplate(1, true, cs.gvk, nil),
				),
				"template6.yaml": fmt.Sprintf("%s%s",
					genTemplate(1, true, cs.gvk, nil),
					genTemplate(1, true, ns.gvk, ownerRefs),
				),
			},
		},
	}

	for _, tc := range testCases {
		engine := NewOwnerRefEngine(&mockEngine{out: tc.input}, restMapper, ownerRefs)
		out, err := engine.Render(&chart.Chart{}, map[string]interface{}{})
		require.NoError(t, err)
		for expectedKey, expectedValue := range tc.expected {
			actualValue, actualKeyExists := out[expectedKey]
			require.True(t, actualKeyExists, "Did not find expected template %q in output", expectedKey)
			require.EqualValues(t, expectedValue, actualValue)
		}
	}
}
