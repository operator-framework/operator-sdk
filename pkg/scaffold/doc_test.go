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
)

func TestDoc(t *testing.T) {
	r, err := NewResource(appApiVersion, appKind)
	if err != nil {
		t.Fatal(err)
	}
	s, buf := setupScaffoldAndWriter()
	err = s.Execute(appConfig, &Doc{Resource: r})
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if docExp != buf.String() {
		diffs := diff(docExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

const docExp = `// Package v1alpha1 contains API Schema definitions for the app v1alpha1 API group
// +k8s:deepcopy-gen=package,register
// +groupName=app.example.com
package v1alpha1
`
