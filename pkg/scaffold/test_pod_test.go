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

func TestPodTest(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig,
		&TestPod{
			Image:            "quay.io/app/operator:v1.0.0",
			TestNamespaceEnv: "TEST_NAMESPACE",
		})
	if err != nil {
		t.Fatalf("failed to execute the scaffold: (%v)", err)
	}

	if testPodExp != buf.String() {
		diffs := diffutil.Diff(testPodExp, buf.String())
		t.Fatalf("expected vs actual differs.\n%v", diffs)
	}
}

const testPodExp = `apiVersion: v1
kind: Pod
metadata:
  name: app-operator-test
spec:
  restartPolicy: Never
  containers:
  - name: app-operator-test
    image: quay.io/app/operator:v1.0.0
    imagePullPolicy: Always
    command: ["/go-test.sh"]
    env:
      - name: TEST_NAMESPACE
        valueFrom:
          fieldRef:
            fieldPath: metadata.namespace
`
