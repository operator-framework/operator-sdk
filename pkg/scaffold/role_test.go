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
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
)

func TestRole(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &Role{})
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if roleExp != buf.String() {
		diffs := diffutil.Diff(roleExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

func TestRoleClusterScoped(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &Role{IsClusterScoped: true})
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if clusterroleExp != buf.String() {
		diffs := diffutil.Diff(clusterroleExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

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
  - endpoints
  - persistentvolumeclaims
  - events
  - configmaps
  - secrets
  verbs:
  - "*"
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
  - "*"
- apiGroups:
  - monitoring.coreos.com
  resources:
  - servicemonitors
  verbs:
  - "get"
  - "create"
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
  - endpoints
  - persistentvolumeclaims
  - events
  - configmaps
  - secrets
  verbs:
  - "*"
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
  - "*"
- apiGroups:
  - monitoring.coreos.com
  resources:
  - servicemonitors
  verbs:
  - "get"
  - "create"
`
