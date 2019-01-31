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

func TestCR(t *testing.T) {
	r, err := NewResource(appApiVersion, appKind)
	if err != nil {
		t.Fatal(err)
	}
	s, buf := setupScaffoldAndWriter()
	err = s.Execute(appConfig, &CR{Resource: r})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if crExp != buf.String() {
		diffs := diffutil.Diff(crExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

func TestCRCustomSpec(t *testing.T) {
	r, err := NewResource(appApiVersion, appKind)
	if err != nil {
		t.Fatal(err)
	}
	s, buf := setupScaffoldAndWriter()
	err = s.Execute(appConfig, &CR{
		Resource: r,
		Spec:     "# Custom spec here\ncustomSize: 6",
	})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if crCustomSpecExp != buf.String() {
		diffs := diffutil.Diff(crCustomSpecExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const crExp = `apiVersion: app.example.com/v1alpha1
kind: AppService
metadata:
  name: example-appservice
spec:
  # Add fields here
  size: 3
`

const crCustomSpecExp = `apiVersion: app.example.com/v1alpha1
kind: AppService
metadata:
  name: example-appservice
spec:
  # Custom spec here
  customSize: 6
`
