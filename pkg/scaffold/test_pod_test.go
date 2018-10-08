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

	"github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func TestPodTest(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig,
		&TestPod{
			Image:            "quay.io/app/operator:v1.0.0",
			TestNamespaceEnv: test.TestNamespaceEnv,
		})
	if err != nil {
		t.Fatalf("expected nil error, got: (%v)", err)
	}

	if testPodExp != buf.String() {
		dmp := diffmatchpatch.New()
		diffs := diffmatchpatch.New().DiffMain(testPodExp, buf.String(), false)
		t.Fatalf("expected vs actual differs. Red text is missing and green text is extra.\n%v", dmp.DiffPrettyText(diffs))
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
