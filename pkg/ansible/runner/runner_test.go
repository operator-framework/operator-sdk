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

package runner

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNewFromWatches(t *testing.T) {
	testCases := []struct {
		name        string
		path        string
		expectedMap map[schema.GroupVersionKind]runner
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
			name:        "error invalid duration",
			path:        "testdata/invalid_duration.yaml",
			shouldError: true,
		},
		{
			name: "valid watches file",
			path: "testdata/valid.yaml",
			expectedMap: map[schema.GroupVersionKind]runner{
				schema.GroupVersionKind{
					Version: "v1alpha1",
					Group:   "app.example.com",
					Kind:    "NoFinalizer",
				}: runner{
					GVK: schema.GroupVersionKind{
						Version: "v1alpha1",
						Group:   "app.example.com",
						Kind:    "NoFinalizer",
					},
					Path:            "/opt/ansible/playbook.yaml",
					reconcilePeriod: time.Second * 2,
				},
				schema.GroupVersionKind{
					Version: "v1alpha1",
					Group:   "app.example.com",
					Kind:    "Playbook",
				}: runner{
					GVK: schema.GroupVersionKind{
						Version: "v1alpha1",
						Group:   "app.example.com",
						Kind:    "Playbook",
					},
					Path: "/opt/ansible/playbook.yaml",
					Finalizer: &Finalizer{
						Name: "finalizer.app.example.com",
						Role: "/opt/ansible/role",
						Vars: map[string]interface{}{"sentinel": "finalizer_running"},
					},
				},
				schema.GroupVersionKind{
					Version: "v1alpha1",
					Group:   "app.example.com",
					Kind:    "Role",
				}: runner{
					GVK: schema.GroupVersionKind{
						Version: "v1alpha1",
						Group:   "app.example.com",
						Kind:    "Role",
					},
					Path: "/opt/ansible/role",
					Finalizer: &Finalizer{
						Name:     "finalizer.app.example.com",
						Playbook: "/opt/ansible/playbook.yaml",
						Vars:     map[string]interface{}{"sentinel": "finalizer_running"},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := NewFromWatches(tc.path)
			if err != nil && !tc.shouldError {
				t.Fatalf("err: %v occurred unexpectedly", err)
			}
			if err != nil && tc.shouldError {
				return
			}
			for k, expectedR := range tc.expectedMap {
				r, ok := m[k]
				if !ok {
					t.Fatalf("did not find expected GVK: %v", k)
				}
				run, ok := r.(*runner)
				if !ok {
					t.Fatalf("here: %#v", r)
				}
				if run.Path != expectedR.Path {
					t.Fatalf("the GVK: %v unexpected path: %v expected path: %v", k, run.Path, expectedR.Path)
				}
				if run.GVK != expectedR.GVK {
					t.Fatalf("the GVK: %v\nunexpected GVK: %#v\nexpected GVK: %#v", k, run.GVK, expectedR.GVK)
				}
				if run.Finalizer != expectedR.Finalizer {
					if run.Finalizer.Name != expectedR.Finalizer.Name || run.Finalizer.Playbook != expectedR.Finalizer.Playbook || run.Finalizer.Role != expectedR.Finalizer.Role || reflect.DeepEqual(run.Finalizer.Vars["sentinel"], expectedR.Finalizer.Vars["sentininel"]) {
						t.Fatalf("the GVK: %v\nunexpected finalizer: %#v\nexpected finalizer: %#v", k, run.Finalizer, expectedR.Finalizer)
					}
				}
				if run.reconcilePeriod != expectedR.reconcilePeriod {
					t.Fatalf("the GVK: %v unexpected reconcile period: %v expected reconcile period: %v", k, run.reconcilePeriod, expectedR.reconcilePeriod)
				}
			}
		})
	}
}
