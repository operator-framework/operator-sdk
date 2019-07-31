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
	"fmt"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/helm"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

func TestGenerateRoleScaffold(t *testing.T) {
	dcs := map[string]*mockRoleDiscoveryClient{
		"upstream": &mockRoleDiscoveryClient{
			serverVersion:   func() (*version.Info, error) { return &version.Info{Major: "1", Minor: "11"}, nil },
			serverResources: func() ([]*metav1.APIResourceList, error) { return simpleResourcesList(), nil },
		},
		"openshift": &mockRoleDiscoveryClient{
			serverVersion:   func() (*version.Info, error) { return &version.Info{Major: "1", Minor: "11+"}, nil },
			serverResources: func() ([]*metav1.APIResourceList, error) { return simpleResourcesList(), nil },
		},
	}

	testCases := []roleScaffoldTestCase{
		{
			name:                   "fallback to default",
			chart:                  failChart(),
			expectSkipDefaultRules: false,
			expectIsClusterScoped:  false,
			expectLenCustomRules:   2,
		},
		{
			name:                   "namespaced manifest",
			chart:                  namespacedChart(),
			expectSkipDefaultRules: true,
			expectIsClusterScoped:  false,
			expectLenCustomRules:   3,
		},
		{
			name:                   "cluster scoped manifest",
			chart:                  clusterScopedChart(),
			expectSkipDefaultRules: true,
			expectIsClusterScoped:  true,
			expectLenCustomRules:   4,
		},
	}

	for _, tc := range testCases {
		for dcName, dc := range dcs {
			testName := fmt.Sprintf("%s %s", dcName, tc.name)
			t.Run(testName, func(t *testing.T) {
				roleScaffold := helm.GenerateRoleScaffold(dc, tc.chart)
				assert.Equal(t, tc.expectSkipDefaultRules, roleScaffold.SkipDefaultRules)
				assert.Equal(t, tc.expectLenCustomRules, len(roleScaffold.CustomRules))
				assert.Equal(t, tc.expectIsClusterScoped, roleScaffold.IsClusterScoped)
			})
		}
	}
}

type mockRoleDiscoveryClient struct {
	serverVersion   func() (*version.Info, error)
	serverResources func() ([]*metav1.APIResourceList, error)
}

func (dc *mockRoleDiscoveryClient) ServerVersion() (*version.Info, error) {
	return dc.serverVersion()
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
			Name: "unknown",
		},
		Templates: []*chart.Template{
			{Name: "broken1.yaml", Data: []byte(`invalid {{ template`)},
		},
	}
}

func namespacedChart() *chart.Chart {
	return &chart.Chart{
		Metadata: &chart.Metadata{
			Name: "namespaced",
		},
		Templates: []*chart.Template{
			{Name: "unknown1.yaml", Data: testUnknownData("unknown1")},
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
		Templates: []*chart.Template{
			{Name: "unknown1.yaml", Data: testUnknownData("unknown1")},
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
