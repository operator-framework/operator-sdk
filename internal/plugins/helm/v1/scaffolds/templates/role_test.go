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

package templates_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/chart"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"

	"github.com/operator-framework/operator-sdk/internal/plugins/helm/v1/scaffolds/templates"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
)

func TestGenerateRoleScaffold(t *testing.T) {
	validDiscoveryClient := &mockRoleDiscoveryClient{
		serverGroupsAndResources: func() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
			return simpleGroupList(), simpleResourcesList(), nil
		},
	}

	brokenDiscoveryClient := &mockRoleDiscoveryClient{
		serverGroupsAndResources: func() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
			return nil, nil, errors.New("no server resources")
		},
	}

	testCases := []roleScaffoldTestCase{
		{
			name:                   "fallback to default with unparsable template",
			chart:                  failChart(),
			expectSkipDefaultRules: false,
			expectLenCustomRules:   3,
		},
		{
			name:                   "skip rule for unknown API",
			chart:                  unknownAPIChart(),
			expectSkipDefaultRules: true,
			expectLenCustomRules:   4,
		},
		{
			name:                   "namespaced manifest",
			chart:                  namespacedChart(),
			expectSkipDefaultRules: true,
			expectLenCustomRules:   4,
		},
		{
			name:                   "cluster scoped manifest",
			chart:                  clusterScopedChart(),
			expectSkipDefaultRules: true,
			expectLenCustomRules:   5,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s with valid discovery client", tc.name), func(t *testing.T) {
			roleScaffold := templates.GenerateRoleScaffold(validDiscoveryClient, tc.chart)
			assert.Equal(t, tc.expectSkipDefaultRules, roleScaffold.SkipDefaultRules)
			assert.Equal(t, tc.expectLenCustomRules, len(roleScaffold.CustomRules))
		})

		t.Run(fmt.Sprintf("%s with broken discovery client", tc.name), func(t *testing.T) {
			roleScaffold := templates.GenerateRoleScaffold(brokenDiscoveryClient, tc.chart)
			assert.Equal(t, false, roleScaffold.SkipDefaultRules)
			assert.Equal(t, 3, len(roleScaffold.CustomRules))
		})
	}
}

type mockRoleDiscoveryClient struct {
	serverGroupsAndResources func() ([]*metav1.APIGroup, []*metav1.APIResourceList, error)
}

func (dc *mockRoleDiscoveryClient) ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error) {
	return dc.serverGroupsAndResources()
}

func simpleGroupList() []*metav1.APIGroup {
	return []*metav1.APIGroup{
		{
			Name: "example",
		},
		{
			Name: "example2",
		},
	}
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

func TestMergeRoleForResource(t *testing.T) {
	clusterRoleFilePath1 := "./testdata/testroles/valid_clusterrole"
	roleFilePath1 := "./testdata/testroles/valid_role1"
	roleFilePath2 := "./testdata/testroles/valid_role2"
	roleFilePath3 := "./testdata/testroles/valid_role3"
	roleFilePath4 := "./testdata/testroles/valid_role4"
	roleFilePath5 := "./testdata/testroles/valid_role5"
	roleFilePath6 := "./testdata/testroles/valid_role6"

	testFiles := map[string]string{
		"./testdata/testroles/valid_clusterrole/config/rbac/role.yaml": clusterRole,
		"./testdata/testroles/valid_role1/config/rbac/role.yaml":       roleFile1,
		"./testdata/testroles/valid_role2/config/rbac/role.yaml":       roleFile2,
		"./testdata/testroles/valid_role3/config/rbac/role.yaml":       roleFile3,
		"./testdata/testroles/valid_role4/config/rbac/role.yaml":       roleFile3,
		"./testdata/testroles/valid_role5/config/rbac/role.yaml":       roleFile3,
		"./testdata/testroles/valid_role6/config/rbac/role.yaml":       roleFile3,
	}

	for path, content := range testFiles {
		assert.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		assert.NoError(t, ioutil.WriteFile(path, []byte(content), fileutil.DefaultFileMode))
		defer remove(filepath.Dir(path))
	}

	testCases := []struct {
		name           string
		absProjectPath string
		r              *resource.Resource
		roleScaffold   *templates.Role
		expError       bool
		existingRole   string
		mergedRole     string
	}{
		{
			name:           "Valid Basic ClusterRole-1",
			absProjectPath: clusterRoleFilePath1,
			mergedRole:     clusterRoleFilePath1 + "/mergedRole.yaml",
			r: &resource.Resource{
				Version:          "v1alpha1",
				Group:            "helm.k8s.io",
				Domain:           "chart",
				Kind:             "Memcached",
				GroupPackageName: "charts.helm.k8s.io",
			},
			roleScaffold: &templates.Role{
				SkipDefaultRules: true,
				CustomRules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps", "secrets"},
						Verbs:     []string{"*"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"events"},
						Verbs:     []string{"create"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"services"},
						Verbs:     []string{"*"},
					},
					{
						APIGroups: []string{"apps"},
						Resources: []string{"statefulsets"},
						Verbs:     []string{"*"},
					},
				},
			},
		},
		{
			name:           "Valid Basic CustomRules-1",
			absProjectPath: roleFilePath1,
			mergedRole:     roleFilePath1 + "/mergedRole.yaml",
			r: &resource.Resource{
				Version:          "v1alpha1",
				Group:            "helm.k8s.io",
				Domain:           "chart",
				Kind:             "Memcached",
				GroupPackageName: "charts.helm.k8s.io",
			},
			roleScaffold: &templates.Role{
				SkipDefaultRules: true,
				CustomRules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps", "secrets"},
						Verbs:     []string{"*"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"events"},
						Verbs:     []string{"create"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"services"},
						Verbs:     []string{"*"},
					},
					{
						APIGroups: []string{"apps"},
						Resources: []string{"statefulsets"},
						Verbs:     []string{"*"},
					},
				},
			},
		},

		{
			name:           "Valid With Options CustomRules-2",
			absProjectPath: roleFilePath2,
			mergedRole:     roleFilePath2 + "/mergedRole.yaml",
			r: &resource.Resource{
				Version:          "v1alpha1",
				Group:            "example.com",
				Domain:           "cache",
				Kind:             "Mykind",
				GroupPackageName: "cache.example.com",
			},
			roleScaffold: &templates.Role{
				SkipDefaultRules: true,
				CustomRules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps", "secrets"},
						Verbs:     []string{"*"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"events"},
						Verbs:     []string{"create"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"serviceaccounts", "services"},
						Verbs:     []string{"*"},
					},
					{
						APIGroups:       []string{"apps"},
						Resources:       []string{"deployments"},
						ResourceNames:   []string{"helm-demo"},
						NonResourceURLs: []string{"/demos"},
						Verbs:           []string{"*"},
					},
					{
						APIGroups:       []string{"apps"},
						Resources:       []string{"deamonsets"},
						Verbs:           []string{"delete"},
						ResourceNames:   []string{"helm-demo"},
						NonResourceURLs: []string{"/demos", "/helm"},
					},
				},
			},
		},
		{
			name:           "Valid and differing APIGroups in CustomRules-3",
			absProjectPath: roleFilePath3,
			mergedRole:     roleFilePath3 + "/mergedRole.yaml",
			r: &resource.Resource{
				Version:          "v1alpha1",
				Group:            "example.com",
				Domain:           "cache",
				Kind:             "Mykind",
				GroupPackageName: "cache.example.com",
			},
			roleScaffold: &templates.Role{
				SkipDefaultRules: true,
				CustomRules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps", "secrets"},
						Verbs:     []string{"*"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"events"},
						Verbs:     []string{"create"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"serviceaccounts", "services"},
						Verbs:     []string{"*"},
					},
					// Testing vars which differ only in APIGroups with existing role.yaml
					{
						APIGroups:       []string{"apps"},
						Resources:       []string{"replicasets", "deployments"},
						ResourceNames:   []string{"helm-demo", "sample"},
						NonResourceURLs: []string{"/demos", "/helm"},
						Verbs:           []string{"create", "get"},
					},
				},
			},
		},
		{
			name:           "Valid and differing ResourceNames in CustomRules-4",
			absProjectPath: roleFilePath4,
			mergedRole:     roleFilePath4 + "/mergedRole.yaml",
			r: &resource.Resource{
				Version:          "v1alpha1",
				Group:            "example.com",
				Domain:           "cache",
				Kind:             "Mykind",
				GroupPackageName: "cache.example.com",
			},
			roleScaffold: &templates.Role{
				SkipDefaultRules: true,
				CustomRules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps", "secrets"},
						Verbs:     []string{"*"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"events"},
						Verbs:     []string{"create"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"serviceaccounts", "services"},
						Verbs:     []string{"*"},
					},
					// Testing vars which differ only in ResourceNames with existing role.yaml
					{
						APIGroups:       []string{"apps", "samples"},
						Resources:       []string{"replicasets", "deployments"},
						ResourceNames:   []string{"helm-demo"},
						NonResourceURLs: []string{"/demos", "/helm"},
						Verbs:           []string{"create", "get"},
					},
				},
			},
		},
		{
			name:           "Valid and differing NonResourceURLs in CustomRule-5",
			absProjectPath: roleFilePath5,
			mergedRole:     roleFilePath5 + "/mergedRole.yaml",
			r: &resource.Resource{
				Version:          "v1alpha1",
				Group:            "example.com",
				Domain:           "cache",
				Kind:             "Mykind",
				GroupPackageName: "cache.example.com",
			},
			roleScaffold: &templates.Role{
				SkipDefaultRules: true,
				CustomRules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps", "secrets"},
						Verbs:     []string{"*"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"events"},
						Verbs:     []string{"create"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"serviceaccounts", "services"},
						Verbs:     []string{"*"},
					},
					// Testing vars which differ only in NonResourceURLs with existing role.yaml
					{
						APIGroups:       []string{"apps", "samples"},
						Resources:       []string{"replicasets", "deployments"},
						ResourceNames:   []string{"helm-demo", "sample"},
						NonResourceURLs: []string{"/demos"},
						Verbs:           []string{"create", "get"},
					},
				},
			},
		},
		{
			name:           "Valid and differing Verbs in CustomRules-6",
			absProjectPath: roleFilePath6,
			mergedRole:     roleFilePath6 + "/mergedRole.yaml",
			r: &resource.Resource{
				Version:          "v1alpha1",
				Group:            "example.com",
				Domain:           "cache",
				Kind:             "Mykind",
				GroupPackageName: "cache.example.com",
			},
			roleScaffold: &templates.Role{
				SkipDefaultRules: true,
				CustomRules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"configmaps", "secrets"},
						Verbs:     []string{"*"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"events"},
						Verbs:     []string{"create"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"serviceaccounts", "services"},
						Verbs:     []string{"*"},
					},
					// Testing vars which differ only in Verbs with existing role.yaml
					{
						APIGroups:       []string{"apps", "samples"},
						Resources:       []string{"replicasets", "deployments"},
						ResourceNames:   []string{"helm-demo", "sample"},
						NonResourceURLs: []string{"/demos", "/helm"},
						Verbs:           []string{"create"},
					},
				},
			},
		},
		{
			name:           "Empty CustomRules",
			expError:       true,
			absProjectPath: "./testdata/testroles/invalid_role",
			mergedRole:     "",
			r: &resource.Resource{
				Version:          "v1alpha1",
				Group:            "example.com",
				Domain:           "cache",
				Kind:             "Mykind",
				GroupPackageName: "app.example.com",
			},
			roleScaffold: &templates.Role{
				SkipDefaultRules: true,
				CustomRules:      []rbacv1.PolicyRule{},
			},
		},
		{
			name:           "Empty role.yaml file",
			expError:       true,
			absProjectPath: "./testdata/testroles/invalid_role",
			mergedRole:     "",
			r: &resource.Resource{
				Version:          "v1alpha1",
				Group:            "example.com",
				Domain:           "cache",
				Kind:             "Mykind",
				GroupPackageName: "app.example.com",
			},
			roleScaffold: &templates.Role{
				SkipDefaultRules: true,
				CustomRules: []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"namespaces"},
						Verbs:     []string{"get"},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := templates.MergeRoleForResource(tc.r, tc.absProjectPath, *tc.roleScaffold)
			if tc.expError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			absFilePath := tc.absProjectPath + "/config/rbac/role.yaml"
			actualMergedRoleYAML, err := ioutil.ReadFile(absFilePath)
			assert.NoError(t, err)
			expectedMergedRoleYAML, err := ioutil.ReadFile(tc.mergedRole)
			assert.NoError(t, err)
			println(t.Name())
			assert.Equal(t, string(expectedMergedRoleYAML), string(actualMergedRoleYAML))
		})
	}
}

// remove removes path from disk. Used in defer statements.
func remove(path string) {
	if err := os.RemoveAll(path); err != nil {
		log.Fatal(err)
	}
}

const clusterRole = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: helm-demo
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
- apiGroups:
  - ""
  resources:
  - configmaps
  - secrets
  verbs:
  - '*'
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
- apiGroups:
  - apps
  resources:
  - deployments
  verbs:
  - '*'
- apiGroups:
  - ""
  resources:
  - persistentvolumeclaims
  - secrets
  - services
  verbs:
  - '*'
- apiGroups:
  - charts.helm.k8s.io
  resources:
  - '*'
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch

`
const roleFile3 = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: helm-demo
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
- apiGroups:
  - ""
  resources:
  - configmaps
  - secrets
  verbs:
  - '*'
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
- apiGroups:
  - ""
  resources:
  - services
  verbs:
  - '*'
- apiGroups:
  - apps
  resources:
  - statefulsets
  verbs:
  - '*'
- apiGroups:
  - charts.helm.k8s.io
  resources:
  - '*'
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
`

const roleFile1 = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: helm-demo
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
- apiGroups:
  - ""
  resources:
  - configmaps
  - secrets
  verbs:
  - '*'
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
- apiGroups:
  - apps
  resources:
  - deployments
  verbs:
  - '*'
- apiGroups:
  - ""
  resources:
  - persistentvolumeclaims
  - secrets
  - services
  verbs:
  - '*'
- apiGroups:
  - charts.helm.k8s.io
  resources:
  - '*'
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch

`
const roleFile2 = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: helm-demo
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
- apiGroups:
  - ""
  resources:
  - configmaps
  - secrets
  verbs:
  - '*'
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create
- apiGroups:
  - ""
  resources:
  - services
  verbs:
  - '*'
- apiGroups:
  - apps
  resources:
  - statefulsets
  verbs:
  - '*'
- apiGroups:
  - charts.helm.k8s.io
  resources:
  - '*'
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
`
