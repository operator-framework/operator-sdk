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

package catalog

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
)

func TestConcatCRD(t *testing.T) {
	buf := &bytes.Buffer{}
	s := &scaffold.Scaffold{
		GetWriter: func(_ string, _ os.FileMode) (io.Writer, error) {
			return buf, nil
		},
	}

	err := s.Execute(&input.Config{},
		&ConcatCRD{
			ConfigFilePath: filepath.Join(testDataDir, scaffold.OLMCatalogDir, CSVConfigYamlFile),
		},
	)
	if err != nil {
		t.Fatalf("Failed to execute the scaffold: (%v)", err)
	}

	if concatCRDExp != buf.String() {
		diffs := diffutil.Diff(concatCRDExp, buf.String())
		t.Fatalf("Expected vs actual differs.\n%v", diffs)
	}
}

const concatCRDExp = `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: apps.example.com
spec:
  group: app.example.com
  names:
    kind: App
    listKind: AppList
    plural: apps
    singular: app
  scope: Namespaced
  version: v1alpha1

---

apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: apps.example.com
spec:
  group: app.example.com
  names:
    kind: App
    listKind: AppList
    plural: apps
    singular: app
  scope: Namespaced
  version: v1alpha2
`
