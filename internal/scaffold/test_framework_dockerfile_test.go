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

func TestTestFrameworkDockerfile(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &TestFrameworkDockerfile{})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if testFrameworkDockerfileExp != buf.String() {
		diffs := diffutil.Diff(testFrameworkDockerfileExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const testFrameworkDockerfileExp = `ARG BASEIMAGE
FROM ${BASEIMAGE}
ADD build/_output/bin/app-operator-test /usr/local/bin/app-operator-test
ARG NAMESPACEDMAN
ADD $NAMESPACEDMAN /namespaced.yaml
ADD build/test-framework/go-test.sh /go-test.sh
`
