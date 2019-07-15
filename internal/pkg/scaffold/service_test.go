// Copyright 2019 The Operator-SDK Authors
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

func TestService(t *testing.T) {
	s, buf := setupScaffoldAndWriter()
	err := s.Execute(appConfig, &Service{})
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if serviceExp != buf.String() {
		diffs := diffutil.Diff(serviceExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const serviceExp = `apiVersion: v1
kind: Service
metadata:
  annotations:
    service.alpha.openshift.io/serving-cert-secret-name: app-operator
  labels:
    name: app-operator
  name: app-operator
spec:
  ports:
  - name: cr-metrics
    port: 9696
    targetPort: cr-metrics
  - name: https-metrics
    port: 9393
    targetPort: https-metrics
  selector:
    name: app-operator
  type: ClusterIP
`
