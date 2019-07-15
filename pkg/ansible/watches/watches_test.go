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

package watches

import (
	"html/template"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestLoad(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Unable to get working director: %v", err)
	}

	validTemplate := struct {
		ValidPlaybook string
		ValidRole     string
	}{
		ValidPlaybook: filepath.Join(cwd, "testdata", "playbook.yml"),
		ValidRole:     filepath.Join(cwd, "testdata", "roles", "role"),
	}

	tmpl, err := template.ParseFiles("testdata/valid.yaml.tmpl")
	if err != nil {
	}
	f, err := os.Create("testdata/valid.yaml")
	if err != nil {
		t.Fatalf("Unable to create valid.yaml: %v", err)
	}
	defer os.Remove("testdata/valid.yaml")
	err = tmpl.Execute(f, validTemplate)
	if err != nil {
		t.Fatalf("Unable to create valid.yaml: %v", err)
		return
	}

	zeroSeconds := time.Duration(0)
	twoSeconds := time.Second * 2
	testCases := []struct {
		name        string
		path        string
		expected    []Watch
		shouldError bool
	}{
		{
			name:        "error duplicate GVK",
			path:        "testdata/duplicate_gvk.yaml",
			shouldError: true,
		},
		{
			name:        "error no file",
			path:        "testdata/please_don't_create_me_gvk.yaml",
			shouldError: true,
		},
		{
			name:        "error invalid yaml",
			path:        "testdata/invalid.yaml",
			shouldError: true,
		},
		{
			name:        "error invalid playbook path",
			path:        "testdata/invalid_playbook_path.yaml",
			shouldError: true,
		},
		{
			name:        "error invalid playbook finalizer path",
			path:        "testdata/invalid_finalizer_playbook_path.yaml",
			shouldError: true,
		},
		{
			name:        "error invalid role path",
			path:        "testdata/invalid_role_path.yaml",
			shouldError: true,
		},
		{
			name:        "error invalid role finalizer path",
			path:        "testdata/invalid_finalizer_role_path.yaml",
			shouldError: true,
		},
		{
			name:        "error invalid finalizer no path/role/vars",
			path:        "testdata/invalid_finalizer_no_vars.yaml",
			shouldError: true,
		},
		{
			name:        "error invalid duration",
			path:        "testdata/invalid_duration.yaml",
			shouldError: true,
		},
		{
			name:        "error invalid status",
			path:        "testdata/invalid_status.yaml",
			shouldError: true,
		},
		{
			name: "valid watches file",
			path: "testdata/valid.yaml",
			expected: []Watch{
				Watch{
					GroupVersionKind: schema.GroupVersionKind{
						Version: "v1alpha1",
						Group:   "app.example.com",
						Kind:    "NoFinalizer",
					},
					Playbook:                    validTemplate.ValidPlaybook,
					ManageStatus:                true,
					ReconcilePeriod:             twoSeconds,
					WatchDependentResources:     true,
					WatchClusterScopedResources: false,
				},
				Watch{
					GroupVersionKind: schema.GroupVersionKind{
						Version: "v1alpha1",
						Group:   "app.example.com",
						Kind:    "Playbook",
					},
					Playbook:                    validTemplate.ValidPlaybook,
					ManageStatus:                true,
					WatchDependentResources:     true,
					WatchClusterScopedResources: false,
					Finalizer: &Finalizer{
						Name: "finalizer.app.example.com",
						Role: validTemplate.ValidRole,
						Vars: map[string]interface{}{"sentinel": "finalizer_running"},
					},
				},
				Watch{
					GroupVersionKind: schema.GroupVersionKind{
						Version: "v1alpha1",
						Group:   "app.example.com",
						Kind:    "WatchClusterScoped",
					},
					Playbook:                    validTemplate.ValidPlaybook,
					ReconcilePeriod:             twoSeconds,
					ManageStatus:                true,
					WatchDependentResources:     true,
					WatchClusterScopedResources: true,
				},
				Watch{
					GroupVersionKind: schema.GroupVersionKind{
						Version: "v1alpha1",
						Group:   "app.example.com",
						Kind:    "NoReconcile",
					},
					Playbook:        validTemplate.ValidPlaybook,
					ReconcilePeriod: zeroSeconds,
					ManageStatus:    true,
				},
				Watch{
					GroupVersionKind: schema.GroupVersionKind{
						Version: "v1alpha1",
						Group:   "app.example.com",
						Kind:    "DefaultStatus",
					},
					Playbook:     validTemplate.ValidPlaybook,
					ManageStatus: true,
				},
				Watch{
					GroupVersionKind: schema.GroupVersionKind{
						Version: "v1alpha1",
						Group:   "app.example.com",
						Kind:    "DisableStatus",
					},
					Playbook:     validTemplate.ValidPlaybook,
					ManageStatus: false,
				},
				Watch{
					GroupVersionKind: schema.GroupVersionKind{
						Version: "v1alpha1",
						Group:   "app.example.com",
						Kind:    "EnableStatus",
					},
					Playbook:     validTemplate.ValidPlaybook,
					ManageStatus: true,
				},
				Watch{
					GroupVersionKind: schema.GroupVersionKind{
						Version: "v1alpha1",
						Group:   "app.example.com",
						Kind:    "Role",
					},
					Role:         validTemplate.ValidRole,
					ManageStatus: true,
					Finalizer: &Finalizer{
						Name:     "finalizer.app.example.com",
						Playbook: validTemplate.ValidPlaybook,
						Vars:     map[string]interface{}{"sentinel": "finalizer_running"},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			watchSlice, err := Load(tc.path)
			if err != nil && !tc.shouldError {
				t.Fatalf("Error occurred unexpectedly: %v", err)
			}
			if err != nil && tc.shouldError {
				return
			}
			for idx, expectedWatch := range tc.expected {
				gvk := expectedWatch.GroupVersionKind
				gotWatch := watchSlice[idx]
				if gotWatch.GroupVersionKind != gvk {
					t.Fatalf("Unexpected GVK: \nunexpected GVK: %#v\nexpected GVK: %#v", gotWatch.GroupVersionKind, gvk)
				}
				// TODO: not clear if this is needed or what the corollary would be without a struct behind an interface
				// run, ok := r.(*runner)
				// if !ok {
				// 	t.Fatalf("Here: %#v", r)
				// }
				if gotWatch.Role != expectedWatch.Role {
					t.Fatalf("The GVK: %v unexpected Role: %v expected Role: %v", gvk, gotWatch.Role, expectedWatch.Role)
				}
				if gotWatch.Playbook != expectedWatch.Playbook {
					t.Fatalf("The GVK: %v unexpected Playbook: %v expected Playbook: %v", gvk, gotWatch.Playbook, expectedWatch.Playbook)
				}
				if gotWatch.ManageStatus != expectedWatch.ManageStatus {
					t.Fatalf("The GVK: %v\nunexpected manageStatus:%#v\nexpected manageStatus: %#v", gvk, gotWatch.ManageStatus, expectedWatch.ManageStatus)
				}
				if gotWatch.Finalizer != expectedWatch.Finalizer {
					if gotWatch.Finalizer.Name != expectedWatch.Finalizer.Name || gotWatch.Finalizer.Playbook != expectedWatch.Finalizer.Playbook || gotWatch.Finalizer.Role != expectedWatch.Finalizer.Role || reflect.DeepEqual(gotWatch.Finalizer.Vars["sentinel"], expectedWatch.Finalizer.Vars["sentininel"]) {
						t.Fatalf("The GVK: %v\nunexpected finalizer: %#v\nexpected finalizer: %#v", gvk, gotWatch.Finalizer, expectedWatch.Finalizer)
					}
				}
				if gotWatch.ReconcilePeriod != expectedWatch.ReconcilePeriod {
					t.Fatalf("The GVK: %v unexpected reconcile period: %v expected reconcile period: %v", gvk, gotWatch.ReconcilePeriod, expectedWatch.ReconcilePeriod)
				}
			}
		})
	}
}
