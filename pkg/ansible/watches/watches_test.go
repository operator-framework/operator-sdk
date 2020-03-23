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
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestNew(t *testing.T) {
	basicGVK := schema.GroupVersionKind{
		Version: "v1alpha1",
		Group:   "app.example.com",
		Kind:    "Example",
	}
	testCases := []struct {
		name           string
		gvk            schema.GroupVersionKind
		role           string
		playbook       string
		vars           map[string]interface{}
		finalizer      *Finalizer
		shouldValidate bool
	}{
		{
			name:           "default invalid watch",
			gvk:            basicGVK,
			shouldValidate: false,
		},
	}
	expectedReconcilePeriod, _ := time.ParseDuration(reconcilePeriodDefault)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			watch := New(tc.gvk, tc.role, tc.playbook, tc.vars, tc.finalizer)
			if watch.GroupVersionKind != tc.gvk {
				t.Fatalf("Unexpected GVK %v expected %v", watch.GroupVersionKind, tc.gvk)
			}
			if watch.MaxRunnerArtifacts != maxRunnerArtifactsDefault {
				t.Fatalf("Unexpected maxRunnerArtifacts %v expected %v", watch.MaxRunnerArtifacts,
					maxRunnerArtifactsDefault)
			}
			if watch.MaxWorkers != maxWorkersDefault {
				t.Fatalf("Unexpected maxWorkers %v expected %v", watch.MaxWorkers, maxWorkersDefault)
			}
			if watch.ReconcilePeriod != expectedReconcilePeriod {
				t.Fatalf("Unexpected reconcilePeriod %v expected %v", watch.ReconcilePeriod,
					expectedReconcilePeriod)
			}
			if watch.ManageStatus != manageStatusDefault {
				t.Fatalf("Unexpected manageStatus %v expected %v", watch.ManageStatus, manageStatusDefault)
			}
			if watch.WatchDependentResources != watchDependentResourcesDefault {
				t.Fatalf("Unexpected watchDependentResources %v expected %v", watch.WatchDependentResources,
					watchDependentResourcesDefault)
			}
			if watch.WatchClusterScopedResources != watchClusterScopedResourcesDefault {
				t.Fatalf("Unexpected watchClusterScopedResources %v expected %v",
					watch.WatchClusterScopedResources, watchClusterScopedResourcesDefault)
			}
			if watch.AnsibleVerbosity != ansibleVerbosityDefault {
				t.Fatalf("Unexpected ansibleVerbosity %v expected %v", watch.AnsibleVerbosity,
					ansibleVerbosityDefault)
			}

			err := watch.Validate()
			if err != nil && tc.shouldValidate {
				t.Fatalf("Watch %v failed validation", watch)
			}
			if err == nil && !tc.shouldValidate {
				t.Fatalf("Watch %v should have failed validation", watch)
			}
		})
	}
}

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
		t.Fatalf("Unable to parse template: %v", err)
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

	validWatches := []Watch{
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
		Watch{
			GroupVersionKind: schema.GroupVersionKind{
				Version: "v1alpha1",
				Group:   "app.example.com",
				Kind:    "FinalizerRole",
			},
			Role:         validTemplate.ValidRole,
			ManageStatus: true,
			Finalizer: &Finalizer{
				Name: "finalizer.app.example.com",
				Vars: map[string]interface{}{"sentinel": "finalizer_running"},
			},
		},
		Watch{
			GroupVersionKind: schema.GroupVersionKind{
				Version: "v1alpha1",
				Group:   "app.example.com",
				Kind:    "MaxWorkersDefault",
			},
			Role:         validTemplate.ValidRole,
			ManageStatus: true,
			MaxWorkers:   1,
		},
		Watch{
			GroupVersionKind: schema.GroupVersionKind{
				Version: "v1alpha1",
				Group:   "app.example.com",
				Kind:    "MaxWorkersIgnored",
			},
			Role:         validTemplate.ValidRole,
			ManageStatus: true,
			MaxWorkers:   1,
		},
		Watch{
			GroupVersionKind: schema.GroupVersionKind{
				Version: "v1alpha1",
				Group:   "app.example.com",
				Kind:    "MaxWorkersEnv",
			},
			Role:         validTemplate.ValidRole,
			ManageStatus: true,
			MaxWorkers:   4,
		},
		Watch{
			GroupVersionKind: schema.GroupVersionKind{
				Version: "v1alpha1",
				Group:   "app.example.com",
				Kind:    "AnsibleVerbosityDefault",
			},
			Role:             validTemplate.ValidRole,
			ManageStatus:     true,
			AnsibleVerbosity: 2,
		},
		Watch{
			GroupVersionKind: schema.GroupVersionKind{
				Version: "v1alpha1",
				Group:   "app.example.com",
				Kind:    "AnsibleVerbosityIgnored",
			},
			Role:             validTemplate.ValidRole,
			ManageStatus:     true,
			AnsibleVerbosity: 2,
		},
		Watch{
			GroupVersionKind: schema.GroupVersionKind{
				Version: "v1alpha1",
				Group:   "app.example.com",
				Kind:    "AnsibleVerbosityEnv",
			},
			Role:             validTemplate.ValidRole,
			ManageStatus:     true,
			AnsibleVerbosity: 4,
		},
		Watch{
			GroupVersionKind: schema.GroupVersionKind{
				Version: "v1alpha1",
				Group:   "app.example.com",
				Kind:    "WatchWithVars",
			},
			Role:         validTemplate.ValidRole,
			ManageStatus: true,
			Vars:         map[string]interface{}{"sentinel": "reconciling"},
		},
		Watch{
			GroupVersionKind: schema.GroupVersionKind{
				Version: "v1alpha1",
				Group:   "app.example.com",
				Kind:    "AnsibleCollectionEnvTest",
			},
			Role:         filepath.Join(cwd, "testdata", "ansible_collections", "nameSpace", "collection", "roles", "someRole"),
			ManageStatus: true,
		},
	}

	testCases := []struct {
		name                                 string
		path                                 string
		maxWorkers                           int
		ansibleVerbosity                     int
		expected                             []Watch
		shouldError                          bool
		shouldSetAnsibleRolePathEnvVar       bool
		shouldSetAnsibleCollectionPathEnvVar bool
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
			name:        "error invalid finalizer whithout name",
			path:        "testdata/invalid_finalizer_whithout_name.yaml",
			shouldError: true,
		},
		{
			name:        "error invalid role path",
			path:        "testdata/invalid_role_path.yaml",
			shouldError: true,
		},
		{
			name:        "error invalid yaml file",
			path:        "testdata/invalid_yaml_file.yaml",
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
			name:        "if collection env var is not set and collection is not installed to the default locations, fail",
			path:        "testdata/invalid_collection.yaml",
			shouldError: true,
		},
		{
			name:                                 "valid watches file",
			path:                                 "testdata/valid.yaml",
			maxWorkers:                           1,
			ansibleVerbosity:                     2,
			shouldSetAnsibleCollectionPathEnvVar: true,
			expected:                             validWatches,
		},
		{
			name:                                 "should load file successfully with ANSIBLE ROLES PATH ENV VAR set",
			path:                                 "testdata/valid.yaml",
			maxWorkers:                           1,
			ansibleVerbosity:                     2,
			shouldSetAnsibleRolePathEnvVar:       true,
			shouldSetAnsibleCollectionPathEnvVar: true,
			expected:                             validWatches,
		},
	}

	os.Setenv("WORKER_MAXWORKERSENV_APP_EXAMPLE_COM", "4")
	defer os.Unsetenv("WORKER_MAXWORKERSENV_APP_EXAMPLE_COM")
	os.Setenv("ANSIBLE_VERBOSITY_ANSIBLEVERBOSITYENV_APP_EXAMPLE_COM", "4")
	defer os.Unsetenv("ANSIBLE_VERBOSITY_ANSIBLEVERBOSITYENV_APP_EXAMPLE_COM")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Test Load with ANSIBLE_ROLES_PATH var
			if tc.shouldSetAnsibleRolePathEnvVar {
				anisbleEnvVar := "path/invalid:/path/invalid/myroles:" + projutil.MustGetwd()
				os.Setenv("ANSIBLE_ROLES_PATH", anisbleEnvVar)
				defer os.Unsetenv("ANSIBLE_ROLES_PATH")
			}
			if tc.shouldSetAnsibleCollectionPathEnvVar {

				ansibleCollectionPathEnv := filepath.Join(projutil.MustGetwd(), "testdata")
				os.Setenv("ANSIBLE_COLLECTIONS_PATH", ansibleCollectionPathEnv)
				defer os.Unsetenv("ANSIBLE_COLLECTIONS_PATH")
			}

			watchSlice, err := Load(tc.path, tc.maxWorkers, tc.ansibleVerbosity)
			if err != nil && !tc.shouldError {
				t.Fatalf("Error occurred unexpectedly: %v", err)
			}
			if err != nil && tc.shouldError {
				return
			}
			// meant to protect from adding test to valid without corresponding check
			if len(tc.expected) != len(watchSlice) {
				t.Fatalf("Unexpected watches length: %v expected: %v", len(watchSlice), len(tc.expected))
			}
			for idx, expectedWatch := range tc.expected {
				gvk := expectedWatch.GroupVersionKind
				gotWatch := watchSlice[idx]
				if gotWatch.GroupVersionKind != gvk {
					t.Fatalf("Unexpected GVK: \nunexpected GVK: %#v\nexpected GVK: %#v",
						gotWatch.GroupVersionKind, gvk)
				}
				if gotWatch.Role != expectedWatch.Role {
					t.Fatalf("The GVK: %v unexpected Role: %v expected Role: %v", gvk, gotWatch.Role,
						expectedWatch.Role)
				}
				if gotWatch.Playbook != expectedWatch.Playbook {
					t.Fatalf("The GVK: %v unexpected Playbook: %v expected Playbook: %v", gvk, gotWatch.Playbook,
						expectedWatch.Playbook)
				}
				if gotWatch.ManageStatus != expectedWatch.ManageStatus {
					t.Fatalf("The GVK: %v\nunexpected manageStatus:%#v\nexpected manageStatus: %#v", gvk,
						gotWatch.ManageStatus, expectedWatch.ManageStatus)
				}
				if gotWatch.Finalizer != expectedWatch.Finalizer {
					if gotWatch.Finalizer.Name != expectedWatch.Finalizer.Name || gotWatch.Finalizer.Playbook !=
						expectedWatch.Finalizer.Playbook || gotWatch.Finalizer.Role !=
						expectedWatch.Finalizer.Role || reflect.DeepEqual(gotWatch.Finalizer.Vars["sentinel"],
						expectedWatch.Finalizer.Vars["sentininel"]) {
						t.Fatalf("The GVK: %v\nunexpected finalizer: %#v\nexpected finalizer: %#v", gvk,
							gotWatch.Finalizer, expectedWatch.Finalizer)
					}
				}
				if gotWatch.ReconcilePeriod != expectedWatch.ReconcilePeriod {
					t.Fatalf("The GVK: %v unexpected reconcile period: %v expected reconcile period: %v", gvk,
						gotWatch.ReconcilePeriod, expectedWatch.ReconcilePeriod)
				}

				if expectedWatch.MaxWorkers == 0 {
					if gotWatch.MaxWorkers != tc.maxWorkers {
						t.Fatalf("Unexpected max workers: %v expected workers: %v", gotWatch.MaxWorkers,
							tc.maxWorkers)
					}
				} else {
					if gotWatch.MaxWorkers != expectedWatch.MaxWorkers {
						t.Fatalf("Unexpected max workers: %v expected workers: %v", gotWatch.MaxWorkers,
							expectedWatch.MaxWorkers)
					}
				}
			}
		})
	}
}

func TestMaxWorkers(t *testing.T) {
	testCases := []struct {
		name          string
		gvk           schema.GroupVersionKind
		defValue      int
		expectedValue int
		setEnv        bool
		envKey        string
		envValue      int
	}{
		{
			name: "no env, use default value",
			gvk: schema.GroupVersionKind{
				Group:   "cache.example.com",
				Version: "v1alpha1",
				Kind:    "MemCacheService",
			},
			defValue:      1,
			expectedValue: 1,
			setEnv:        false,
			envKey:        "WORKER_MEMCACHESERVICE_CACHE_EXAMPLE_COM",
		},
		{
			name: "invalid env, use default value",
			gvk: schema.GroupVersionKind{
				Group:   "cache.example.com",
				Version: "v1alpha1",
				Kind:    "MemCacheService",
			},
			defValue:      1,
			expectedValue: 1,
			setEnv:        true,
			envKey:        "WORKER_MEMCACHESERVICE_CACHE_EXAMPLE_COM",
			envValue:      0,
		},
		{
			name: "env set to 3, expect 3",
			gvk: schema.GroupVersionKind{
				Group:   "cache.example.com",
				Version: "v1alpha1",
				Kind:    "MemCacheService",
			},
			defValue:      1,
			expectedValue: 3,
			setEnv:        true,
			envKey:        "WORKER_MEMCACHESERVICE_CACHE_EXAMPLE_COM",
			envValue:      3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Unsetenv(tc.envKey)
			if tc.setEnv {
				os.Setenv(tc.envKey, strconv.Itoa(tc.envValue))
			}
			workers := getMaxWorkers(tc.gvk, tc.defValue)
			if tc.expectedValue != workers {
				t.Fatalf("Unexpected MaxWorkers: %v expected MaxWorkers: %v", workers, tc.expectedValue)
			}
		})
	}
}

func TestAnsibleVerbosity(t *testing.T) {
	testCases := []struct {
		name          string
		gvk           schema.GroupVersionKind
		defValue      int
		expectedValue int
		setEnv        bool
		envKey        string
		envValue      int
	}{
		{
			name: "no env, use default value",
			gvk: schema.GroupVersionKind{
				Group:   "cache.example.com",
				Version: "v1alpha1",
				Kind:    "MemCacheService",
			},
			defValue:      1,
			expectedValue: 1,
			setEnv:        false,
			envKey:        "ANSIBLE_VERBOSITY_MEMCACHESERVICE_CACHE_EXAMPLE_COM",
		},
		{
			name: "invalid env, lt 0, use default value",
			gvk: schema.GroupVersionKind{
				Group:   "cache.example.com",
				Version: "v1alpha1",
				Kind:    "MemCacheService",
			},
			defValue:      1,
			expectedValue: 1,
			setEnv:        true,
			envKey:        "ANSIBLE_VERBOSITY_MEMCACHESERVICE_CACHE_EXAMPLE_COM",
			envValue:      -1,
		},
		{
			name: "invalid env, gt 7, use default value",
			gvk: schema.GroupVersionKind{
				Group:   "cache.example.com",
				Version: "v1alpha1",
				Kind:    "MemCacheService",
			},
			defValue:      1,
			expectedValue: 1,
			setEnv:        true,
			envKey:        "ANSIBLE_VERBOSITY_MEMCACHESERVICE_CACHE_EXAMPLE_COM",
			envValue:      8,
		},
		{
			name: "env set to 3, expect 3",
			gvk: schema.GroupVersionKind{
				Group:   "cache.example.com",
				Version: "v1alpha1",
				Kind:    "MemCacheService",
			},
			defValue:      1,
			expectedValue: 3,
			setEnv:        true,
			envKey:        "ANSIBLE_VERBOSITY_MEMCACHESERVICE_CACHE_EXAMPLE_COM",
			envValue:      3,
		},
		{
			name: "boundary test 0",
			gvk: schema.GroupVersionKind{
				Group:   "cache.example.com",
				Version: "v1alpha1",
				Kind:    "MemCacheService",
			},
			defValue:      1,
			expectedValue: 0,
			setEnv:        true,
			envKey:        "ANSIBLE_VERBOSITY_MEMCACHESERVICE_CACHE_EXAMPLE_COM",
			envValue:      0,
		},
		{
			name: "boundary test 7",
			gvk: schema.GroupVersionKind{
				Group:   "cache.example.com",
				Version: "v1alpha1",
				Kind:    "MemCacheService",
			},
			defValue:      1,
			expectedValue: 7,
			setEnv:        true,
			envKey:        "ANSIBLE_VERBOSITY_MEMCACHESERVICE_CACHE_EXAMPLE_COM",
			envValue:      7,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Unsetenv(tc.envKey)
			if tc.setEnv {
				os.Setenv(tc.envKey, strconv.Itoa(tc.envValue))
			}
			verbosity := getAnsibleVerbosity(tc.gvk, tc.defValue)
			if tc.expectedValue != verbosity {
				t.Fatalf("Unexpected Verbosity: %v expected Verbosity: %v", verbosity, tc.expectedValue)
			}
		})
	}
}

// Test the func getPossibleRolePaths.
func TestGetPossibleRolePaths(t *testing.T) {
	// Mock default Full Path based in the current directory
	rolesPath := filepath.Join(projutil.MustGetwd(), "roles")
	home, _ := os.UserHomeDir()

	type args struct {
		path           string
		rolesEnv       string
		collectionsEnv string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "check the current dir for a role name",
			args: args{
				path: "Foo",
			},
			want: []string{filepath.Join(rolesPath, "Foo")},
		},
		{
			name: "check the current dir for a relative path",
			args: args{
				path: "relative/Foo",
			},
			want: []string{filepath.Join(rolesPath, "relative/Foo")},
		},
		{
			name: "check all paths in ANSIBLE_ROLES_PATH env var",
			args: args{
				rolesEnv: "relative:nested/relative:/and/abs",
				path:     "Foo",
			},
			want: []string{
				filepath.Join(rolesPath, "Foo"),
				filepath.Join("relative", "Foo"),
				filepath.Join("relative", "roles", "Foo"),
				filepath.Join("nested/relative", "Foo"),
				filepath.Join("nested/relative", "roles", "Foo"),
				filepath.Join("/and/abs", "Foo"),
				filepath.Join("/and/abs", "roles", "Foo"),
			},
		},
		{
			name: "Check for roles inside default collection locations when given fqcn",
			args: args{
				path: "myNS.myCol.myRole",
			},
			want: []string{
				filepath.Join(rolesPath, "myNS.myCol.myRole"),
				filepath.Join("/usr/share/ansible/collections", "ansible_collections", "myNS", "myCol", "roles", "myRole"),
				filepath.Join(home, ".ansible/collections", "ansible_collections", "myNS", "myCol", "roles", "myRole"),
			},
		},
		{
			name: "Check for roles inside ANSIBLE_COLLECTIONS_PATH locations when set and given path is fqcn",
			args: args{
				path:           "myNS.myCol.myRole",
				collectionsEnv: "/my/collections/",
			},
			want: []string{
				filepath.Join(rolesPath, "myNS.myCol.myRole"),
				filepath.Join("/my/collections/", "ansible_collections", "myNS", "myCol", "roles", "myRole"),
				// Note: Defaults are not checked when the env variable is set
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if len(tt.args.rolesEnv) > 0 {
				os.Setenv("ANSIBLE_ROLES_PATH", tt.args.rolesEnv)
				defer os.Unsetenv("ANSIBLE_ROLES_PATH")
			}
			if len(tt.args.collectionsEnv) > 0 {
				os.Setenv("ANSIBLE_COLLECTIONS_PATH", tt.args.collectionsEnv)
				defer os.Unsetenv("ANSIBLE_COLLECTIONS_PATH")
			}

			allPathsToCheck := getPossibleRolePaths(tt.args.path)
			sort.Strings(tt.want)
			sort.Strings(allPathsToCheck)
			if !reflect.DeepEqual(allPathsToCheck, tt.want) {
				t.Errorf("Unexpected paths returned")
				fmt.Println("Returned:")
				for i, path := range allPathsToCheck {
					fmt.Println(i, path)
				}
				fmt.Println("Wanted:")
				for i, path := range tt.want {
					fmt.Println(i, path)
				}
			}
		})
	}
}
