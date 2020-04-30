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

package watches

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
)

func TestLoadReader(t *testing.T) {
	trueVal, falseVal := true, false
	testCases := []struct {
		name          string
		data          string
		env           map[string]string
		expectWatches []Watch
		expectErr     bool
	}{
		{
			name: "valid",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
  watchDependentResources: false
  overrideValues:
    key: value
`,
			expectWatches: []Watch{
				{
					GroupVersionKind:        schema.GroupVersionKind{Group: "mygroup", Version: "v1alpha1", Kind: "MyKind"},
					ChartDir:                "../../../internal/scaffold/helm/testdata/testcharts/test-chart",
					WatchDependentResources: &falseVal,
					OverrideValues:          map[string]string{"key": "value"},
				},
			},
			expectErr: false,
		},
		{
			name: "valid with override expansion",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
  watchDependentResources: false
  overrideValues:
    key: $MY_VALUE
`,
			env: map[string]string{"MY_VALUE": "value"},
			expectWatches: []Watch{
				{
					GroupVersionKind:        schema.GroupVersionKind{Group: "mygroup", Version: "v1alpha1", Kind: "MyKind"},
					ChartDir:                "../../../internal/scaffold/helm/testdata/testcharts/test-chart",
					WatchDependentResources: &falseVal,
					OverrideValues:          map[string]string{"key": "value"},
				},
			},
			expectErr: false,
		},
		{
			name: "multiple gvk",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyFirstKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
- group: mygroup
  version: v1alpha1
  kind: MySecondKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
`,
			expectWatches: []Watch{
				{
					GroupVersionKind:        schema.GroupVersionKind{Group: "mygroup", Version: "v1alpha1", Kind: "MyFirstKind"},
					ChartDir:                "../../../internal/scaffold/helm/testdata/testcharts/test-chart",
					WatchDependentResources: &trueVal,
				},
				{
					GroupVersionKind:        schema.GroupVersionKind{Group: "mygroup", Version: "v1alpha1", Kind: "MySecondKind"},
					ChartDir:                "../../../internal/scaffold/helm/testdata/testcharts/test-chart",
					WatchDependentResources: &trueVal,
				},
			},
			expectErr: false,
		},
		{
			name: "duplicate gvk",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
`,
			expectErr: true,
		},
		{
			name: "no version",
			data: `---
- group: mygroup
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
`,
			expectErr: true,
		},
		{
			name: "no kind",
			data: `---
- group: mygroup
  version: v1alpha1
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
`,
			expectErr: true,
		},
		{
			name: "bad chart path",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: nonexistent/path/to/chart
`,
			expectErr: true,
		},
		{
			name: "invalid overrides",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
  overrideValues:
    key1:
		key2: value
`,
			expectErr: true,
		},
		{
			name: "invalid yaml",
			data: `---
foo: bar
`,
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("Failed to set environment variable %q: %v", k, err)
				}
			}

			watchesData := bytes.NewBufferString(tc.data)
			watches, err := LoadReader(watchesData)
			if !tc.expectErr && err != nil {
				t.Fatalf("Expected no error; got error: %v", err)
			} else if tc.expectErr && err == nil {
				t.Fatalf("Expected error; got no error")
			}
			assert.Equal(t, tc.expectWatches, watches)

			for k := range tc.env {
				if err := os.Unsetenv(k); err != nil {
					t.Fatalf("Failed to unset environment variable %q: %v", k, err)
				}
			}
		})
	}
}

func TestLoad(t *testing.T) {
	falseVal := false
	testCases := []struct {
		name          string
		data          string
		env           map[string]string
		expectWatches []Watch
		expectErr     bool
	}{
		{
			name: "valid",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
  watchDependentResources: false
  overrideValues:
    key: value
`,
			expectWatches: []Watch{
				{
					GroupVersionKind:        schema.GroupVersionKind{Group: "mygroup", Version: "v1alpha1", Kind: "MyKind"},
					ChartDir:                "../../../internal/scaffold/helm/testdata/testcharts/test-chart",
					WatchDependentResources: &falseVal,
					OverrideValues:          map[string]string{"key": "value"},
				},
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				if err := os.Setenv(k, v); err != nil {
					t.Fatalf("Failed to set environment variable %q: %v", k, err)
				}
			}

			f, err := ioutil.TempFile("", "osdk-test-load")
			if err != nil {
				t.Fatalf("Failed to create temporary watches file: %v", err)
			}
			defer removeFile(t, f)
			if _, err := f.WriteString(tc.data); err != nil {
				t.Fatalf("Failed to write temporary watches file: %v", err)
			}
			watches, err := Load(f.Name())
			if !tc.expectErr && err != nil {
				t.Fatalf("Expected no error; got error: %v", err)
			} else if tc.expectErr && err == nil {
				t.Fatalf("Expected error; got no error")
			}
			assert.Equal(t, tc.expectWatches, watches)

			for k := range tc.env {
				if err := os.Unsetenv(k); err != nil {
					t.Fatalf("Failed to unset environment variable %q: %v", k, err)
				}
			}
		})
	}
}

func TestAppend(t *testing.T) {
	trueVal := true
	testCases := []struct {
		name          string
		data          string
		watch         Watch
		expectErr     bool
		expectWatches string
	}{
		{
			name: "empty_existing",
			watch: Watch{
				GroupVersionKind: schema.GroupVersionKind{
					Group: "mygroup", Version: "v1alpha1", Kind: "MyNewKind",
				},
				ChartDir:                "helm-charts/test",
				WatchDependentResources: &trueVal,
				OverrideValues:          map[string]string{"key": "value"},
			},
			expectErr: false,
			expectWatches: `---
- group: mygroup
  version: v1alpha1
  kind: MyNewKind
  chart: helm-charts/test
  watchDependentResources: true
  overrideValues:
    "key": "value"
`,
		},
		{
			name: "empty_minimal_watch",
			watch: Watch{
				GroupVersionKind: schema.GroupVersionKind{
					Group: "mygroup", Version: "v1alpha1", Kind: "MyNewKind",
				},
				ChartDir: "helm-charts/test",
			},
			expectErr: false,
			expectWatches: `---
- group: mygroup
  version: v1alpha1
  kind: MyNewKind
  chart: helm-charts/test
`,
		},
		{
			name: "append_all_fields",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
  watchDependentResources: false
  overrideValues:
    "key": "value"
`,
			watch: Watch{
				GroupVersionKind: schema.GroupVersionKind{
					Group: "mygroup", Version: "v1alpha1", Kind: "MyNewKind",
				},
				ChartDir:                "helm-charts/test",
				WatchDependentResources: &trueVal,
				OverrideValues:          map[string]string{"key": "value"},
			},
			expectErr: false,
			expectWatches: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
  watchDependentResources: false
  overrideValues:
    "key": "value"
- group: mygroup
  version: v1alpha1
  kind: MyNewKind
  chart: helm-charts/test
  watchDependentResources: true
  overrideValues:
    "key": "value"
`,
		},
		{
			name: "duplicate_error",
			data: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
  watchDependentResources: false
  overrideValues:
    "key": "value"
`,
			watch: Watch{
				GroupVersionKind: schema.GroupVersionKind{
					Group: "mygroup", Version: "v1alpha1", Kind: "MyKind",
				},
				ChartDir: "helm-charts/test",
			},
			expectErr: true,
		},
		{
			name: "invalid_yaml_error",
			data: `---
group: mygroup
version: v1alpha1
kind: MyKind
chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
watchDependentResources: false
overrideValues:
  "key": "value"
`,
			watch: Watch{
				GroupVersionKind: schema.GroupVersionKind{
					Group: "mygroup", Version: "v1alpha1", Kind: "MyNewKind",
				},
				ChartDir: "helm-charts/test",
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := Append(bytes.NewBufferString(tc.data), tc.watch)
			if !tc.expectErr && err != nil {
				t.Fatalf("Expected no error; got error: %v", err)
			} else if tc.expectErr && err == nil {
				t.Fatalf("Expected error; got no error")
			}
			assert.Equal(t, tc.expectWatches, string(data))
		})
	}
}

func TestUpdateForResource(t *testing.T) {
	resource, err := scaffold.NewResource("mygroup/v1alpha1", "MyNewKind")
	if err != nil {
		t.Fatal("Invalid resource: %w", err)
	}
	testCases := []struct {
		name            string
		initialData     string
		resource        *scaffold.Resource
		chartName       string
		expectWatches   string
		expectErr       bool
		skipInitialFile bool
	}{
		{
			name:            "non-existent",
			resource:        resource,
			chartName:       "test",
			skipInitialFile: true,
			expectErr:       false,
			expectWatches: `---
- group: mygroup
  version: v1alpha1
  kind: MyNewKind
  chart: helm-charts/test
`,
		},
		{
			name:      "empty",
			resource:  resource,
			chartName: "test",
			expectErr: false,
			expectWatches: `---
- group: mygroup
  version: v1alpha1
  kind: MyNewKind
  chart: helm-charts/test
`,
		},
		{
			name:      "existing",
			resource:  resource,
			chartName: "test",
			initialData: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
  watchDependentResources: false
  overrideValues:
    "key": "value"
`,
			expectErr: false,
			expectWatches: `---
- group: mygroup
  version: v1alpha1
  kind: MyKind
  chart: ../../../internal/scaffold/helm/testdata/testcharts/test-chart
  watchDependentResources: false
  overrideValues:
    "key": "value"
- group: mygroup
  version: v1alpha1
  kind: MyNewKind
  chart: helm-charts/test
`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			wf, err := ioutil.TempFile("", "osdk-test-append-to-file")
			if err != nil {
				t.Fatalf("Error creating temporary watches file: %v", err)
			}
			defer func() {
				if err := os.Remove(wf.Name()); err != nil {
					t.Fatalf("Error removing temporary watches file: %v", err)
				}
			}()
			if err := wf.Close(); err != nil {
				t.Fatalf("Error closing temporary watches file: %v", err)
			}
			if tc.skipInitialFile {
				if err := os.Remove(wf.Name()); err != nil {
					t.Fatalf("Error removing temporary watches file: %v", err)
				}
			} else {
				if err := ioutil.WriteFile(wf.Name(), []byte(tc.initialData), fileutil.DefaultFileMode); err != nil {
					t.Fatalf("Error writing test initialData to temporary watches file: %v", err)
				}
			}

			err = UpdateForResource(wf.Name(), tc.resource, tc.chartName)
			if !tc.expectErr && err != nil {
				t.Fatalf("Expected no error; got error: %v", err)
			} else if tc.expectErr && err == nil {
				t.Fatalf("Expected error; got no error")
			}

			watchesData, err := ioutil.ReadFile(wf.Name())
			if err != nil {
				t.Fatalf("Error reading temporary watches file: %v", err)
			}

			assert.Equal(t, tc.expectWatches, string(watchesData))
		})
	}
}

// remove removes path from disk. Used in defer statements.
func removeFile(t *testing.T, f *os.File) {
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(f.Name()); err != nil {
		t.Fatal(err)
	}
}
