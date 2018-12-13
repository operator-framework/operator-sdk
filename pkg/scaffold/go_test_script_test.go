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

func TestGoTestScript(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &GoTestScript{})
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if goTestScriptExp != buf.String() {
		diffs := diffutil.Diff(goTestScriptExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

const goTestScriptExp = `#!/bin/sh

app-operator-test -test.parallel=1 -test.failfast -root=/ -kubeconfig=incluster -namespacedMan=namespaced.yaml -test.v
`
