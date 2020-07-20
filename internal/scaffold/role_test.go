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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	rbacv1 "k8s.io/api/rbac/v1"
)

func TestRole(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &Role{})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if roleExp != buf.String() {
		diffs := diffutil.Diff(roleExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

func TestRoleClusterScoped(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &Role{IsClusterScoped: true})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if clusterroleExp != buf.String() {
		diffs := diffutil.Diff(clusterroleExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

func TestRoleCustomRules(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &Role{
		SkipDefaultRules: true,
		CustomRules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"policy"},
				Resources: []string{"poddisruptionbudgets"},
				Verbs: []string{
					"create",
					"delete",
					"get",
					"list",
					"patch",
					"update",
					"watch",
				},
			},
			{
				APIGroups: []string{"rbac.authorization.k8s.io"},
				Resources: []string{"roles", "rolebindings"},
				Verbs:     []string{"get", "list", "watch"},
			},
		}})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if roleCustomRulesExp != buf.String() {
		diffs := diffutil.Diff(roleCustomRulesExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
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
		"./testdata/testroles/valid_clusterrole/deploy/role.yaml": clusterRole,
		"./testdata/testroles/valid_role1/deploy/role.yaml":       roleFile1,
		"./testdata/testroles/valid_role2/deploy/role.yaml":       roleFile2,
		"./testdata/testroles/valid_role3/deploy/role.yaml":       roleFile3,
		"./testdata/testroles/valid_role4/deploy/role.yaml":       roleFile3,
		"./testdata/testroles/valid_role5/deploy/role.yaml":       roleFile3,
		"./testdata/testroles/valid_role6/deploy/role.yaml":       roleFile3,
	}

	for path, content := range testFiles {
		assert.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
		assert.NoError(t, ioutil.WriteFile(path, []byte(content), fileutil.DefaultFileMode))
		defer remove(filepath.Dir(path))
	}

	testCases := []struct {
		name           string
		absProjectPath string
		r              *Resource
		roleScaffold   *Role
		expError       bool
		existingRole   string
		mergedRole     string
	}{
		{
			name:           "Valid Basic ClusterRole-1",
			absProjectPath: clusterRoleFilePath1,
			mergedRole:     clusterRoleFilePath1 + "/mergedRole.yaml",
			r: &Resource{
				APIVersion: "charts.helm.k8s.io/v1alpha1",
				Kind:       "Memcached",
				FullGroup:  "charts.helm.k8s.io",
				LowerKind:  "memcached",
				Resource:   "memcacheds",
			},
			roleScaffold: &Role{
				IsClusterScoped:  false,
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
			r: &Resource{
				APIVersion: "charts.helm.k8s.io/v1alpha1",
				Kind:       "Memcached",
				FullGroup:  "charts.helm.k8s.io",
				LowerKind:  "memcached",
				Resource:   "memcacheds",
			},
			roleScaffold: &Role{
				IsClusterScoped:  false,
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
			r: &Resource{
				APIVersion: "cache.example.com/v1alpha1",
				Kind:       "Mykind",
				FullGroup:  "cache.example.com",
				LowerKind:  "mykind",
				Resource:   "mykinds",
			},
			roleScaffold: &Role{
				IsClusterScoped:  false,
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
			r: &Resource{
				APIVersion: "cache.example.com/v1alpha1",
				Kind:       "Mykind",
				FullGroup:  "cache.example.com",
				LowerKind:  "mykind",
				Resource:   "mykinds",
			},
			roleScaffold: &Role{
				IsClusterScoped:  false,
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
			r: &Resource{
				APIVersion: "cache.example.com/v1alpha1",
				Kind:       "Mykind",
				FullGroup:  "cache.example.com",
				LowerKind:  "mykind",
				Resource:   "mykinds",
			},
			roleScaffold: &Role{
				IsClusterScoped:  false,
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
			r: &Resource{
				APIVersion: "cache.example.com/v1alpha1",
				Kind:       "Mykind",
				FullGroup:  "cache.example.com",
				LowerKind:  "mykind",
				Resource:   "mykinds",
			},
			roleScaffold: &Role{
				IsClusterScoped:  false,
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
			r: &Resource{
				APIVersion: "cache.example.com/v1alpha1",
				Kind:       "Mykind",
				FullGroup:  "cache.example.com",
				LowerKind:  "mykind",
				Resource:   "mykinds",
			},
			roleScaffold: &Role{
				IsClusterScoped:  false,
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
			name:           "Invalid ClusterRole",
			expError:       true,
			absProjectPath: roleFilePath1,
			r: &Resource{
				APIVersion: "charts.helm.k8s.io/v1alpha1",
				Kind:       "Nginxingress",
				FullGroup:  "charts.helm.k8s.io",
				LowerKind:  "nginx-ingress",
				Resource:   "nginxingresses",
			},
			roleScaffold: &Role{
				IsClusterScoped:  true,
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
				},
			},
		},
		{
			name:           "Empty CustomRules",
			expError:       true,
			absProjectPath: "./testdata/testroles/invalid_role",
			mergedRole:     "",
			r: &Resource{
				APIVersion: "cache.example.com/v1alpha1",
				Kind:       "Mykind",
				FullGroup:  "app.example.com",
			},
			roleScaffold: &Role{
				IsClusterScoped:  false,
				SkipDefaultRules: true,
				CustomRules:      []rbacv1.PolicyRule{},
			},
		},
		{
			name:           "Empty role.yaml file",
			expError:       true,
			absProjectPath: "./testdata/testroles/invalid_role",
			mergedRole:     "",
			r: &Resource{
				APIVersion: "cache.example.com/v1alpha1",
				Kind:       "Mykind",
				FullGroup:  "app.example.com",
			},
			roleScaffold: &Role{
				IsClusterScoped:  false,
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
			err := MergeRoleForResource(tc.r, tc.absProjectPath, *tc.roleScaffold)
			if tc.expError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			absFilePath := tc.absProjectPath + "/deploy/role.yaml"
			actualMergedRoleYAML, err := ioutil.ReadFile(absFilePath)
			assert.NoError(t, err)
			expectedMergedRoleYAML, err := ioutil.ReadFile(tc.mergedRole)
			assert.NoError(t, err)

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
  - monitoring.coreos.com
  resources:
  - servicemonitors
  verbs:
  - get
  - create
- apiGroups:
  - apps
  resourceNames:
  - helm-demo
  resources:
  - deployments/finalizers
  verbs:
  - update
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
- apiGroups:
  - apps
  resources:
  - replicasets
  - deployments
  verbs:
  - get
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
kind: Role
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
  - monitoring.coreos.com
  resources:
  - servicemonitors
  verbs:
  - get
  - create
- apiGroups:
  - apps
  resourceNames:
  - helm-demo
  resources:
  - deployments/finalizers
  verbs:
  - update
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
- apiGroups:
  - apps
  - samples
  nonResourceURLs:
  - /demos
  - /helm
  resourceNames:
  - helm-demo
  - sample 
  resources:
  - replicasets
  - deployments
  verbs:
  - create
  - get
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
kind: Role
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
  - monitoring.coreos.com
  resources:
  - servicemonitors
  verbs:
  - get
  - create
- apiGroups:
  - apps
  resourceNames:
  - helm-demo
  resources:
  - deployments/finalizers
  verbs:
  - update
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
- apiGroups:
  - apps
  resources:
  - replicasets
  - deployments
  verbs:
  - get
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
kind: Role
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
  - monitoring.coreos.com
  resources:
  - servicemonitors
  verbs:
  - get
  - create
- apiGroups:
  - apps
  resourceNames:
  - helm-demo
  resources:
  - deployments/finalizers
  verbs:
  - update
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
- apiGroups:
  - apps
  resources:
  - replicasets
  - deployments
  verbs:
  - get
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

const roleExp = `kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: app-operator
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - services
  - services/finalizers
  - endpoints
  - persistentvolumeclaims
  - events
  - configmaps
  - secrets
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - apps
  resources:
  - deployments
  - daemonsets
  - replicasets
  - statefulsets
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - monitoring.coreos.com
  resources:
  - servicemonitors
  verbs:
  - "get"
  - "create"
- apiGroups:
  - apps
  resources:
  - deployments/finalizers
  resourceNames:
  - app-operator
  verbs:
  - "update"
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
- apiGroups:
  - apps
  resources:
  - replicasets
  - deployments
  verbs:
  - get
`

const clusterroleExp = `kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: app-operator
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - services
  - services/finalizers
  - endpoints
  - persistentvolumeclaims
  - events
  - configmaps
  - secrets
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - apps
  resources:
  - deployments
  - daemonsets
  - replicasets
  - statefulsets
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - monitoring.coreos.com
  resources:
  - servicemonitors
  verbs:
  - "get"
  - "create"
- apiGroups:
  - apps
  resources:
  - deployments/finalizers
  resourceNames:
  - app-operator
  verbs:
  - "update"
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
- apiGroups:
  - apps
  resources:
  - replicasets
  - deployments
  verbs:
  - get
`

const roleCustomRulesExp = `kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: app-operator
rules:
- verbs:
  - "create"
  - "delete"
  - "get"
  - "list"
  - "patch"
  - "update"
  - "watch"
  apiGroups:
  - "policy"
  resources:
  - "poddisruptionbudgets"
- verbs:
  - "get"
  - "list"
  - "watch"
  apiGroups:
  - "rbac.authorization.k8s.io"
  resources:
  - "roles"
  - "rolebindings"
- apiGroups:
  - monitoring.coreos.com
  resources:
  - servicemonitors
  verbs:
  - "get"
  - "create"
- apiGroups:
  - apps
  resources:
  - deployments/finalizers
  resourceNames:
  - app-operator
  verbs:
  - "update"
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
- apiGroups:
  - apps
  resources:
  - replicasets
  - deployments
  verbs:
  - get
`
