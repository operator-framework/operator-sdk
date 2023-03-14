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
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/operator-framework/operator-sdk/internal/ansible/watches"
)

func checkCmdFunc(t *testing.T, cmdFunc cmdFuncType, playbook, role string, verbosity int) {
	ident := "test"
	inputDirPath := "/test/path"
	maxArtifacts := 1
	var expectedCmd, gotCmd *exec.Cmd
	switch {
	case playbook != "":
		expectedCmd = playbookCmdFunc(playbook)(ident, inputDirPath, maxArtifacts, verbosity)
	case role != "":
		expectedCmd = roleCmdFunc(role)(ident, inputDirPath, maxArtifacts, verbosity)
	}

	gotCmd = cmdFunc(ident, inputDirPath, maxArtifacts, verbosity)

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
		name             string
		gvk              schema.GroupVersionKind
		playbook         string
		role             string
		vars             map[string]interface{}
		finalizer        *watches.Finalizer
		desiredObjectKey string
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
				Name:     "operator.example.com/finalizer",
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
				Name: "operator.example.com/finalizer",
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
				Name: "operator.example.com/finalizer",
				Vars: map[string]interface{}{
					"state": "absent",
				},
			},
		},
		{
			name: "basic runner with playbook, vars + finalizer vars",
			gvk: schema.GroupVersionKind{
				Group:   "operator.example.com",
				Version: "v1alpha1",
				Kind:    "Example",
			},
			playbook: validPlaybook,
			vars: map[string]interface{}{
				"type": "this",
			},
			finalizer: &watches.Finalizer{
				Name: "operator.example.com/finalizer",
				Vars: map[string]interface{}{
					"state": "absent",
				},
			},
		},
		{
			name: "basic runner with a dash in the group name",
			gvk: schema.GroupVersionKind{
				Group:   "operator-with-dash.example.com",
				Version: "v1alpha1",
				Kind:    "Example",
			},
			playbook:         validPlaybook,
			desiredObjectKey: "_operator_with_dash_example_com_example",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testWatch := watches.New(tc.gvk, tc.role, tc.playbook, tc.vars, tc.finalizer)

			testRunner, err := New(*testWatch, "")
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

			// check that the group + kind are properly formatted into a parameter
			if tc.desiredObjectKey != "" {
				parameters := testRunnerStruct.makeParameters(&unstructured.Unstructured{})
				if _, ok := parameters[tc.desiredObjectKey]; !ok {
					t.Fatalf("Did not find expected objKey %v in parameters %+v", tc.desiredObjectKey, parameters)
				}

			}

			if testRunnerStruct.GVK != testWatch.GroupVersionKind {
				t.Fatalf("Unexpected GVK %v expected GVK %v", testRunnerStruct.GVK, testWatch.GroupVersionKind)
			}

			if testRunnerStruct.maxRunnerArtifacts != testWatch.MaxRunnerArtifacts {
				t.Fatalf("Unexpected maxRunnerArtifacts %v expected maxRunnerArtifacts %v",
					testRunnerStruct.maxRunnerArtifacts, testWatch.MaxRunnerArtifacts)
			}

			// Check the cmdFunc
			checkCmdFunc(t, testRunnerStruct.cmdFunc, testWatch.Playbook, testWatch.Role, testWatch.AnsibleVerbosity)

			// Check finalizer
			if testRunnerStruct.Finalizer != testWatch.Finalizer {
				t.Fatalf("Unexpected finalizer %v expected finalizer %v", testRunnerStruct.Finalizer,
					testWatch.Finalizer)
			}

			if testWatch.Finalizer != nil {
				if testRunnerStruct.Finalizer.Name != testWatch.Finalizer.Name {
					t.Fatalf("Unexpected finalizer name %v expected finalizer name %v",
						testRunnerStruct.Finalizer.Name, testWatch.Finalizer.Name)
				}

				if len(testWatch.Finalizer.Vars) == 0 {
					checkCmdFunc(t, testRunnerStruct.cmdFunc, testWatch.Finalizer.Playbook, testWatch.Finalizer.Role,
						testWatch.AnsibleVerbosity)
				} else {
					// when finalizer vars is set the finalizerCmdFunc should be the same as the cmdFunc
					checkCmdFunc(t, testRunnerStruct.finalizerCmdFunc, testWatch.Playbook, testWatch.Role,
						testWatch.AnsibleVerbosity)
				}
			}
		})
	}
}

func TestAnsibleVerbosityString(t *testing.T) {
	testCases := []struct {
		verbosity      int
		expectedString string
	}{
		{verbosity: -1, expectedString: ""},
		{verbosity: 0, expectedString: ""},
		{verbosity: 1, expectedString: "-v"},
		{verbosity: 2, expectedString: "-vv"},
		{verbosity: 7, expectedString: "-vvvvvvv"},
	}

	for _, tc := range testCases {
		gotString := ansibleVerbosityString(tc.verbosity)
		if tc.expectedString != gotString {
			t.Fatalf("Unexpected string %v for  expected %v from verbosity %v", gotString, tc.expectedString, tc.verbosity)
		}
	}
}

func TestMakeParameters(t *testing.T) {
	var (
		inputSpec = "testKey"
	)

	testCases := []struct {
		name               string
		inputParams        unstructured.Unstructured
		expectedSafeParams interface{}
	}{
		{
			name: "should mark values passed as string unsafe",
			inputParams: unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						inputSpec: "testVal",
					},
				},
			},
			expectedSafeParams: map[string]interface{}{
				"__ansible_unsafe": "testVal",
			},
		},
		{
			name: "should not mark integers unsafe",
			inputParams: unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						inputSpec: 3,
					},
				},
			},
			expectedSafeParams: 3,
		},
		{
			name: "should recursively mark values in dictionary as unsafe",
			inputParams: unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						inputSpec: map[string]interface{}{
							"testsubKey1": "val1",
							"testsubKey2": "val2",
						},
					},
				},
			},
			expectedSafeParams: map[string]interface{}{
				"testsubKey1": map[string]interface{}{
					"__ansible_unsafe": "val1",
				},
				"testsubKey2": map[string]interface{}{
					"__ansible_unsafe": "val2",
				},
			},
		},
		{
			name: "should recursively mark values in list as unsafe",
			inputParams: unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						inputSpec: []interface{}{
							"testVal1",
							"testVal2",
						},
					},
				},
			},
			expectedSafeParams: []interface{}{
				map[string]interface{}{
					"__ansible_unsafe": "testVal1",
				},
				map[string]interface{}{
					"__ansible_unsafe": "testVal2",
				},
			},
		},
		{
			name: "should recursively mark values in list/dict as unsafe",
			inputParams: unstructured.Unstructured{
				Object: map[string]interface{}{
					"spec": map[string]interface{}{
						inputSpec: []interface{}{
							"testVal1",
							"testVal2",
							map[string]interface{}{
								"testVal3": 3,
								"testVal4": "__^&{__)",
							},
						},
					},
				},
			},
			expectedSafeParams: []interface{}{
				map[string]interface{}{
					"__ansible_unsafe": "testVal1",
				},
				map[string]interface{}{
					"__ansible_unsafe": "testVal2",
				},
				map[string]interface{}{
					"testVal3": 3,
					"testVal4": map[string]interface{}{
						"__ansible_unsafe": "__^&{__)",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		testRunner := runner{
			markUnsafe: true,
		}
		parameters := testRunner.makeParameters(&tc.inputParams)

		val, ok := parameters[inputSpec]
		if !ok {
			t.Fatalf("Error occurred, value %s in spec is missing", inputSpec)
		} else {
			eq := reflect.DeepEqual(val, tc.expectedSafeParams)
			if !eq {
				t.Errorf("Error occurred, parameters %v are not marked unsafe", val)
			}
		}
	}
}
