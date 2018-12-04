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

func TestAddToScheme(t *testing.T) {
	r, err := NewResource(appApiVersion, appKind)
	if err != nil {
		t.Fatal(err)
	}
	s, buf := setupScaffoldAndWriter()
	err = s.Execute(appConfig, &AddToScheme{Resource: r})
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if addtoschemeExp != buf.String() {
		diffs := diffutil.Diff(addtoschemeExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

const addtoschemeExp = `package apis

import (
	"github.com/example-inc/app-operator/pkg/apis/app/v1alpha1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1alpha1.SchemeBuilder.AddToScheme)
}
`
