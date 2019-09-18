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
	"fmt"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/operator-framework/operator-sdk/pkg/ansible/watches"
)

func playbookCmd(path string, ident string, inputDirPath string, maxArtifacts int) *exec.Cmd {
	return exec.Command("ansible-runner", "-vv", "--rotate-artifacts", fmt.Sprintf("%v", maxArtifacts), "-p", path, "-i", ident, "run", inputDirPath)
}

func roleCmd(path string, ident string, inputDirPath string, maxArtifacts int) *exec.Cmd {
	rolePath, roleName := filepath.Split(path)
	return exec.Command("ansible-runner", "-vv", "--rotate-artifacts", fmt.Sprintf("%v", maxArtifacts), "--role", roleName, "--roles-path", rolePath, "--hosts", "localhost", "-i", ident, "run", inputDirPath)
}

type cmdFuncType func(ident, inputDirPath string, maxArtifacts int) *exec.Cmd

func checkCmdFunc(t *testing.T, cmdFunc cmdFuncType, role, playbook string, watchesMaxArtifacts, runnerMaxArtifacts int) {
	ident := "test"
	inputDirPath := "/test/path"
	var path string
	var expectedCmd, gotCmd *exec.Cmd
	switch {
	case playbook != "":
		path = playbook
		expectedCmd = playbookCmd(path, ident, inputDirPath, watchesMaxArtifacts)
	case role != "":
		path = role
		expectedCmd = roleCmd(path, ident, inputDirPath, watchesMaxArtifacts)
	}

	gotCmd = cmdFunc(ident, inputDirPath, runnerMaxArtifacts)

	if expectedCmd.Path != gotCmd.Path {
		t.Fatalf("Unexpected cmd path %v expected cmd path %v", gotCmd.Path, expectedCmd.Path)
	}

	if !reflect.DeepEqual(expectedCmd.Args, gotCmd.Args) {
		t.Fatalf("Unexpected cmd args %v expected cmd args %v", gotCmd.Args, expectedCmd.Args)
	}
}

func TestNew(t *testing.T) {
	testCases := []struct {
		name  string
		watch watches.Watch
	}{
		{
			name: "basic runner with playbook",
			watch: watches.Watch{
				GroupVersionKind: schema.GroupVersionKind{
					Group:   "operator.example.com",
					Version: "v1alpha1",
					Kind:    "BasicPlaybook",
				},
				Playbook:                    "/opt/example/playbook.yml",
				MaxRunnerArtifacts:          watches.MaxRunnerArtifactsDefault,
				ReconcilePeriod:             watches.ReconcilePeriodDurationDefault,
				ManageStatus:                watches.ManageStatusDefault,
				WatchDependentResources:     watches.WatchDependentResourcesDefault,
				WatchClusterScopedResources: watches.WatchClusterScopedResourcesDefault,
				Finalizer:                   nil,
			},
		},
		{
			name: "basic runner with role",
			watch: watches.Watch{
				GroupVersionKind: schema.GroupVersionKind{
					Group:   "operator.example.com",
					Version: "v1alpha1",
					Kind:    "BasicPlaybook",
				},
				Role:                        "/opt/example/roles/example_role",
				MaxRunnerArtifacts:          watches.MaxRunnerArtifactsDefault,
				ReconcilePeriod:             watches.ReconcilePeriodDurationDefault,
				ManageStatus:                watches.ManageStatusDefault,
				WatchDependentResources:     watches.WatchDependentResourcesDefault,
				WatchClusterScopedResources: watches.WatchClusterScopedResourcesDefault,
				Finalizer:                   nil,
			},
		},
		{
			name: "basic runner with playbook + finalizer playbook",
			watch: watches.Watch{
				GroupVersionKind: schema.GroupVersionKind{
					Group:   "operator.example.com",
					Version: "v1alpha1",
					Kind:    "BasicPlaybook",
				},
				Playbook:                    "/opt/example/playbook.yml",
				MaxRunnerArtifacts:          watches.MaxRunnerArtifactsDefault,
				ReconcilePeriod:             watches.ReconcilePeriodDurationDefault,
				ManageStatus:                watches.ManageStatusDefault,
				WatchDependentResources:     watches.WatchDependentResourcesDefault,
				WatchClusterScopedResources: watches.WatchClusterScopedResourcesDefault,
				Finalizer: &watches.Finalizer{
					Name:     "example.finalizer.com",
					Playbook: "/opt/example/finalizer.yml",
				},
			},
		},
		{
			name: "basic runner with role + finalizer role",
			watch: watches.Watch{
				GroupVersionKind: schema.GroupVersionKind{
					Group:   "operator.example.com",
					Version: "v1alpha1",
					Kind:    "BasicPlaybook",
				},
				Role:                        "/opt/example/roles/example_role",
				MaxRunnerArtifacts:          watches.MaxRunnerArtifactsDefault,
				ReconcilePeriod:             watches.ReconcilePeriodDurationDefault,
				ManageStatus:                watches.ManageStatusDefault,
				WatchDependentResources:     watches.WatchDependentResourcesDefault,
				WatchClusterScopedResources: watches.WatchClusterScopedResourcesDefault,
				Finalizer: &watches.Finalizer{
					Name: "example.finalizer.com",
					Role: "/opt/example/roles/finalizer_role",
				},
			},
		},
		{
			name: "basic runner with playbook + finalizer vars",
			watch: watches.Watch{
				GroupVersionKind: schema.GroupVersionKind{
					Group:   "operator.example.com",
					Version: "v1alpha1",
					Kind:    "BasicPlaybook",
				},
				Playbook:                    "/opt/example/playbook.yml",
				MaxRunnerArtifacts:          watches.MaxRunnerArtifactsDefault,
				ReconcilePeriod:             watches.ReconcilePeriodDurationDefault,
				ManageStatus:                watches.ManageStatusDefault,
				WatchDependentResources:     watches.WatchDependentResourcesDefault,
				WatchClusterScopedResources: watches.WatchClusterScopedResourcesDefault,
				Finalizer: &watches.Finalizer{
					Name: "example.finalizer.com",
					Vars: map[string]interface{}{
						"state": "absent",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testRunner, err := New(tc.watch)
			if err != nil {
				t.Fatalf("Error occurred unexpectedly: %v", err)
			}
			testRunnerStruct, ok := testRunner.(*runner)
			if !ok {
				t.Fatalf("Error occurred unexpectedly: %v", err)
			}

			switch {
			case tc.watch.Playbook != "":
				if testRunnerStruct.Path != tc.watch.Playbook {
					t.Fatalf("Unexpected path %v expected path %v", testRunnerStruct.Path, tc.watch.Playbook)
				}
			case tc.watch.Role != "":
				if testRunnerStruct.Path != tc.watch.Role {
					t.Fatalf("Unexpected path %v expected path %v", testRunnerStruct.Path, tc.watch.Role)
				}
			}

			if testRunnerStruct.GVK != tc.watch.GroupVersionKind {
				t.Fatalf("Unexpected GVK %v expected GVK %v", testRunnerStruct.GVK, tc.watch.GroupVersionKind)
			}

			if testRunnerStruct.maxRunnerArtifacts != tc.watch.MaxRunnerArtifacts {
				t.Fatalf("Unexpected maxRunnerArtifacts %v expected maxRunnerArtifacts %v", testRunnerStruct.maxRunnerArtifacts, tc.watch.MaxRunnerArtifacts)
			}

			// Check the cmdFunc
			checkCmdFunc(t, testRunnerStruct.cmdFunc, tc.watch.Role, tc.watch.Playbook, tc.watch.MaxRunnerArtifacts, testRunnerStruct.maxRunnerArtifacts)

			// Check finalizer
			if testRunnerStruct.Finalizer != tc.watch.Finalizer {
				t.Fatalf("Unexpected finalizer %v expected finalizer %v", testRunnerStruct.Finalizer, tc.watch.Finalizer)
			}

			if tc.watch.Finalizer != nil {
				if testRunnerStruct.Finalizer.Name != tc.watch.Finalizer.Name {
					t.Fatalf("Unexpected finalizer name %v expected finalizer name %v", testRunnerStruct.Finalizer.Name, tc.watch.Finalizer.Name)
				}

				if len(tc.watch.Finalizer.Vars) == 0 {
					checkCmdFunc(t, testRunnerStruct.finalizerCmdFunc, tc.watch.Finalizer.Role, tc.watch.Finalizer.Playbook, tc.watch.MaxRunnerArtifacts, testRunnerStruct.maxRunnerArtifacts)
				} else {
					// when finalizer vars is set the finalizerCmdFunc should be the same as the cmdFunc
					checkCmdFunc(t, testRunnerStruct.finalizerCmdFunc, tc.watch.Role, tc.watch.Playbook, tc.watch.MaxRunnerArtifacts, testRunnerStruct.maxRunnerArtifacts)
				}
			}

		})
	}
}
