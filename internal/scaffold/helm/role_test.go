// Copyright 2019 The Operator-SDK Authors
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

package helm_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/scaffold/helm"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateRoleScaffold(t *testing.T) {
	validDiscoveryClient := &mockRoleDiscoveryClient{
		serverResources: func() ([]*metav1.APIResourceList, error) { return simpleResourcesList(), nil },
	}

	brokenDiscoveryClient := &mockRoleDiscoveryClient{
		serverResources: func() ([]*metav1.APIResourceList, error) { return nil, errors.New("no server resources") },
	}

	testCases := []roleScaffoldTestCase{
		{
			name:                   "fallback to default with unparsable template",
			chart:                  failChart(),
			expectSkipDefaultRules: false,
			expectIsClusterScoped:  false,
			expectLenCustomRules:   3,
		},
		{
			name:                   "skip rule for unknown API",
			chart:                  unknownAPIChart(),
			expectSkipDefaultRules: true,
			expectIsClusterScoped:  false,
			expectLenCustomRules:   4,
		},
		{
			name:                   "namespaced manifest",
			chart:                  namespacedChart(),
			expectSkipDefaultRules: true,
			expectIsClusterScoped:  false,
			expectLenCustomRules:   4,
		},
		{
			name:                   "cluster scoped manifest",
			chart:                  clusterScopedChart(),
			expectSkipDefaultRules: true,
			expectIsClusterScoped:  true,
			expectLenCustomRules:   5,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s with valid discovery client", tc.name), func(t *testing.T) {
			roleScaffold := helm.GenerateRoleScaffold(validDiscoveryClient, tc.chart)
			assert.Equal(t, tc.expectSkipDefaultRules, roleScaffold.SkipDefaultRules)
			assert.Equal(t, tc.expectLenCustomRules, len(roleScaffold.CustomRules))
			assert.Equal(t, tc.expectIsClusterScoped, roleScaffold.IsClusterScoped)
		})

		t.Run(fmt.Sprintf("%s with broken discovery client", tc.name), func(t *testing.T) {
			roleScaffold := helm.GenerateRoleScaffold(brokenDiscoveryClient, tc.chart)
			assert.Equal(t, false, roleScaffold.SkipDefaultRules)
			assert.Equal(t, 3, len(roleScaffold.CustomRules))
			assert.Equal(t, false, roleScaffold.IsClusterScoped)
		})
	}
}

type mockRoleDiscoveryClient struct {
	serverResources func() ([]*metav1.APIResourceList, error)
}

func (dc *mockRoleDiscoveryClient) ServerResources() ([]*metav1.APIResourceList, error) {
	return dc.serverResources()
}

func simpleResourcesList() []*metav1.APIResourceList {
	return []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
					Name:       "namespaces",
					Kind:       "Namespace",
					Namespaced: false,
				},
				{
					Name:       "pods",
					Kind:       "Pod",
					Namespaced: true,
				},
			},
		},
	}
}

type roleScaffoldTestCase struct {
	name                   string
	chart                  *chart.Chart
	expectSkipDefaultRules bool
	expectIsClusterScoped  bool
	expectLenCustomRules   int
}

func failChart() *chart.Chart {
	return &chart.Chart{
		Metadata: &chart.Metadata{
			Name: "broken",
		},
		Templates: []*chart.File{
			{Name: "broken1.yaml", Data: []byte(`invalid {{ template`)},
		},
	}
}

func unknownAPIChart() *chart.Chart {
	return &chart.Chart{
		Metadata: &chart.Metadata{
			Name: "unknown",
		},
		Templates: []*chart.File{
			{Name: "unknown1.yaml", Data: testUnknownData("unknown1")},
			{Name: "pod1.yaml", Data: testPodData("pod1")},
		},
	}
}

func namespacedChart() *chart.Chart {
	return &chart.Chart{
		Metadata: &chart.Metadata{
			Name: "namespaced",
		},
		Templates: []*chart.File{
			{Name: "pod1.yaml", Data: testPodData("pod1")},
			{Name: "pod2.yaml", Data: testPodData("pod2")},
		},
	}
}

func clusterScopedChart() *chart.Chart {
	return &chart.Chart{
		Metadata: &chart.Metadata{
			Name: "clusterscoped",
		},
		Templates: []*chart.File{
			{Name: "pod1.yaml", Data: testPodData("pod1")},
			{Name: "pod2.yaml", Data: testPodData("pod2")},
			{Name: "ns1.yaml", Data: testNamespaceData("ns1")},
			{Name: "ns2.yaml", Data: testNamespaceData("ns2")},
		},
	}
}

func testUnknownData(name string) []byte {
	return []byte(fmt.Sprintf(`apiVersion: my-test-unknown.unknown.com/v1alpha1
kind: UnknownKind
metadata:
  name: %s`, name),
	)
}

func testPodData(name string) []byte {
	return []byte(fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
spec:
  containers:
  - name: test
    image: test`, name),
	)
}

func testNamespaceData(name string) []byte {
	return []byte(fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s`, name),
	)
}
