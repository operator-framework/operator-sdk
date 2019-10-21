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
	"os"
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
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Unable to get working director: %v", err)
	}
	validPlaybook := filepath.Join(cwd, "testdata", "playbook.yml")
	validRole := filepath.Join(cwd, "testdata", "roles", "role")
	testCases := []struct {
		name      string
		gvk       schema.GroupVersionKind
		playbook  string
		role      string
		finalizer *watches.Finalizer
	}{
		{
			name: "basic runner with playbook",
			gvk: schema.GroupVersionKind{
				Group:   "operator.example.com",
				Version: "v1alpha1",
				Kind:    "Example",
			},
			playbook: validPlaybook,
		},
		{
			name: "basic runner with role",
			gvk: schema.GroupVersionKind{
				Group:   "operator.example.com",
				Version: "v1alpha1",
				Kind:    "Example",
			},
			role: validRole,
		},
		{
			name: "basic runner with playbook + finalizer playbook",
			gvk: schema.GroupVersionKind{
				Group:   "operator.example.com",
				Version: "v1alpha1",
				Kind:    "Example",
			},
			playbook: validPlaybook,
			finalizer: &watches.Finalizer{
				Name:     "example.finalizer.com",
				Playbook: validPlaybook,
			},
		},
		{
			name: "basic runner with role + finalizer role",
			gvk: schema.GroupVersionKind{
				Group:   "operator.example.com",
				Version: "v1alpha1",
				Kind:    "Example",
			},
			role: validRole,
			finalizer: &watches.Finalizer{
				Name: "example.finalizer.com",
				Role: validRole,
			},
		},
		{
			name: "basic runner with playbook + finalizer vars",
			gvk: schema.GroupVersionKind{
				Group:   "operator.example.com",
				Version: "v1alpha1",
				Kind:    "Example",
			},
			playbook: validPlaybook,
			finalizer: &watches.Finalizer{
				Name: "example.finalizer.com",
				Vars: map[string]interface{}{
					"state": "absent",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testWatch := watches.New(tc.gvk, tc.role, tc.playbook, tc.finalizer)

			testRunner, err := New(*testWatch)
			if err != nil {
				t.Fatalf("Error occurred unexpectedly: %v", err)
			}
			testRunnerStruct, ok := testRunner.(*runner)
			if !ok {
				t.Fatalf("Error occurred unexpectedly: %v", err)
			}

			switch {
			case testWatch.Playbook != "":
				if testRunnerStruct.Path != testWatch.Playbook {
					t.Fatalf("Unexpected path %v expected path %v", testRunnerStruct.Path, testWatch.Playbook)
				}
			case testWatch.Role != "":
				if testRunnerStruct.Path != testWatch.Role {
					t.Fatalf("Unexpected path %v expected path %v", testRunnerStruct.Path, testWatch.Role)
				}
			}

			if testRunnerStruct.GVK != testWatch.GroupVersionKind {
				t.Fatalf("Unexpected GVK %v expected GVK %v", testRunnerStruct.GVK, testWatch.GroupVersionKind)
			}

			if testRunnerStruct.maxRunnerArtifacts != testWatch.MaxRunnerArtifacts {
				t.Fatalf("Unexpected maxRunnerArtifacts %v expected maxRunnerArtifacts %v", testRunnerStruct.maxRunnerArtifacts, testWatch.MaxRunnerArtifacts)
			}

			// Check the cmdFunc
			checkCmdFunc(t, testRunnerStruct.cmdFunc, testWatch.Role, testWatch.Playbook, testWatch.MaxRunnerArtifacts, testRunnerStruct.maxRunnerArtifacts)

			// Check finalizer
			if testRunnerStruct.Finalizer != testWatch.Finalizer {
				t.Fatalf("Unexpected finalizer %v expected finalizer %v", testRunnerStruct.Finalizer, testWatch.Finalizer)
			}

			if testWatch.Finalizer != nil {
				if testRunnerStruct.Finalizer.Name != testWatch.Finalizer.Name {
					t.Fatalf("Unexpected finalizer name %v expected finalizer name %v", testRunnerStruct.Finalizer.Name, testWatch.Finalizer.Name)
				}

				if len(testWatch.Finalizer.Vars) == 0 {
					checkCmdFunc(t, testRunnerStruct.finalizerCmdFunc, testWatch.Finalizer.Role, testWatch.Finalizer.Playbook, testWatch.MaxRunnerArtifacts, testRunnerStruct.maxRunnerArtifacts)
				} else {
					// when finalizer vars is set the finalizerCmdFunc should be the same as the cmdFunc
					checkCmdFunc(t, testRunnerStruct.finalizerCmdFunc, testWatch.Role, testWatch.Playbook, testWatch.MaxRunnerArtifacts, testRunnerStruct.maxRunnerArtifacts)
				}
			}
		})
	}
}
