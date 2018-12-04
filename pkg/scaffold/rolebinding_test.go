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

func TestRoleBinding(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &RoleBinding{})
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if rolebindingExp != buf.String() {
		diffs := diffutil.Diff(rolebindingExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

func TestRoleBindingClusterScoped(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &RoleBinding{IsClusterScoped: true})
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if clusterrolebindingExp != buf.String() {
		diffs := diffutil.Diff(clusterrolebindingExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

const rolebindingExp = `kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: app-operator
subjects:
- kind: ServiceAccount
  name: app-operator
roleRef:
  kind: Role
  name: app-operator
  apiGroup: rbac.authorization.k8s.io
`

const clusterrolebindingExp = `kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: app-operator
subjects:
- kind: ServiceAccount
  name: app-operator
  # Replace this with the namespace the operator is deployed in.
  namespace: REPLACE_NAMESPACE
roleRef:
  kind: ClusterRole
  name: app-operator
  apiGroup: rbac.authorization.k8s.io
`
