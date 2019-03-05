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
	"strings"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/fileutil"

	"github.com/stretchr/testify/assert"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
	"github.com/stretchr/testify/require"
)

func TestCRDGoProject(t *testing.T) {
	r, err := NewResource("cache.example.com/v1alpha1", "Memcached")
	if err != nil {
		t.Fatal(err)
	}
	s, buf := setupScaffoldAndWriter()
	absPath, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// Set the project and repo paths to {abs}/test/test-framework, which
	// contains pkg/apis for the memcached-operator.
	tfDir := filepath.Join("test", "test-framework-memcached")
	pkgIdx := strings.Index(absPath, "pkg")
	cfg := &input.Config{
		Repo:           filepath.Join(absPath[strings.Index(absPath, "github.com"):pkgIdx], tfDir),
		AbsProjectPath: filepath.Join(absPath[:pkgIdx], tfDir),
		ProjectName:    tfDir,
	}
	if err := os.Chdir(cfg.AbsProjectPath); err != nil {
		t.Fatal(err)
	}
	defer func() { os.Chdir(absPath) }()
	err = s.Execute(cfg, &CRD{
		Input:        input.Input{Path: filepath.Join(tfDir, "cache_v1alpha1_memcached.yaml")},
		Resource:     r,
		IsOperatorGo: true,
	})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if crdGoExp != buf.String() {
		diffs := diffutil.Diff(crdGoExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const crdGoExp = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: memcacheds.cache.example.com
spec:
  group: cache.example.com
  names:
    kind: Memcached
    listKind: MemcachedList
    plural: memcacheds
    singular: memcached
  scope: Namespaced
  subresources:
    status: {}
  validation:
    openAPIV3Schema:
      properties:
        apiVersion:
          type: string
        kind:
          type: string
        metadata:
          type: object
        spec:
          properties:
            size:
              format: int32
              type: integer
          required:
          - size
          type: object
        status:
          properties:
            nodes:
              items:
                type: string
              type: array
          required:
          - nodes
          type: object
  version: v1alpha1
  versions:
  - name: v1alpha1
    served: true
    storage: true
`

const namespacedRole = `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  creationTimestamp: null
  name: memcached-operator
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - services
  - endpoints
  - persistentvolumeclaims
  - events
  - configmaps
  - secrets
  verbs:
  - '*'
- apiGroups:
  - apps
  resources:
  - deployments
  - daemonsets
  - replicasets
  - statefulsets
  verbs:
  - '*'
- apiGroups:
  - cache.example.com
  resources:
  - '*'
  verbs:
  - '*'
`

const clusterRole = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  creationTimestamp: null
  name: podset-cluster-operator
rules:
- apiGroups:
  - ""
  resources:
  - pods
  - services
  - endpoints
  - persistentvolumeclaims
  - events
  - configmaps
  - secrets
  verbs:
  - '*'
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
- apiGroups:
  - apps
  resources:
  - deployments
  - daemonsets
  - replicasets
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
  - app.example.com
  resources:
  - '*'
  verbs:
  - '*'
`

func TestGetScopeForResource(t *testing.T) {

	absPath, err := os.Getwd()
	require.NoError(t, err)
	defer func() { os.Chdir(absPath) }()

	t.Run("namespaced scope", func(t *testing.T) {
		// given
		tmpDir, err := ioutil.TempDir("", "operatorsdk-test")
		defer func() { os.Remove(tmpDir) }()
		require.NoError(t, err)
		deployDir := filepath.Join(tmpDir, DeployDir)
		os.Mkdir(deployDir, os.ModeDir+os.ModePerm)
		os.Chdir(tmpDir)
		err = ioutil.WriteFile(filepath.Join(deployDir, "role.yaml"), []byte(namespacedRole), fileutil.DefaultFileMode)
		// when
		scope, err := getScopeForResource()
		require.NoError(t, err)
		// then
		assert.Equal(t, apiextv1beta1.NamespaceScoped, scope)
	})

	t.Run("clustered scope", func(t *testing.T) {
		// given
		tmpDir, err := ioutil.TempDir("", "operatorsdk-test")
		defer func() { os.Remove(tmpDir) }()
		require.NoError(t, err)
		os.Chdir(tmpDir)
		deployDir := filepath.Join(tmpDir, DeployDir)
		os.Mkdir(deployDir, os.ModeDir+os.ModePerm)
		err = ioutil.WriteFile(filepath.Join(deployDir, "role.yaml"), []byte(clusterRole), fileutil.DefaultFileMode)
		// when
		scope, err := getScopeForResource()
		require.NoError(t, err)
		// then
		assert.Equal(t, apiextv1beta1.ClusterScoped, scope)

	})
}

func TestCRDNonGoProject(t *testing.T) {
	r, err := NewResource(appApiVersion, appKind)
	if err != nil {
		t.Fatal(err)
	}
	s, buf := setupScaffoldAndWriter()
	err = s.Execute(appConfig, &CRD{Resource: r})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if crdNonGoExp != buf.String() {
		diffs := diffutil.Diff(crdNonGoExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const crdNonGoExp = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: appservices.app.example.com
spec:
  group: app.example.com
  names:
    kind: AppService
    listKind: AppServiceList
    plural: appservices
    singular: appservice
  scope: Namespaced
  subresources:
    status: {}
  version: v1alpha1
  versions:
  - name: v1alpha1
    served: true
    storage: true
`
