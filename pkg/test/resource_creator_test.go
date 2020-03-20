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

package test

import (
	"os"
	"testing"

	"k8s.io/client-go/kubernetes/fake"
)

const (
	OperatorNamespaceEnv = "TEST_OPERATOR_NAMESPACE"
	WatchNamespaceEnv    = "TEST_WATCH_NAMESPACE"
)

var fakeNamespacedManPath string = "fakePath"

func TestGetOperatorNamespace(t *testing.T) {
	Global = &Framework{
		NamespacedManPath: &fakeNamespacedManPath,
		KubeClient:        fake.NewSimpleClientset(),
	}

	t.Run("should create a non-empty new Operator Namespace when OperatorNamespace env is not set",
		func(t *testing.T) {
			Global.OperatorNamespace = ""
			ctx := NewContext(t)
			operatorNamespace, err := ctx.GetOperatorNamespace()
			assertNoError(t, err)
			if len(operatorNamespace) <= 0 {
				t.Errorf("Expected non-empty operatorNamespace")
			}
		})
	t.Run("should return Operator Namespace specified by OperatorNamespace Env", func(t *testing.T) {
		operatorNamespace := "test-operator-namespae"
		Global.OperatorNamespace = operatorNamespace
		os.Setenv(TestOperatorNamespaceEnv, operatorNamespace)
		defer func() {
			Global.OperatorNamespace = ""
			os.Unsetenv(OperatorNamespaceEnv)
		}()

		ctx := NewContext(t)
		got, err := ctx.GetOperatorNamespace()
		assertNoError(t, err)
		if got != operatorNamespace {
			t.Errorf("Expected %v, got %v", operatorNamespace, got)
		}
	})
	t.Run("should return non-empty new Operator Namespace, when OperatorNamespace Env = \"\"",
		func(t *testing.T) {
			operatorNamespace := ""
			Global.OperatorNamespace = operatorNamespace
			os.Setenv(TestOperatorNamespaceEnv, operatorNamespace)
			defer func() {
				Global.OperatorNamespace = ""
				os.Unsetenv(OperatorNamespaceEnv)
			}()

			ctx := NewContext(t)
			got, err := ctx.GetOperatorNamespace()
			assertNoError(t, err)
			if len(got) <= 0 {
				t.Errorf("Expected non-empty operatorNamespace")
			}
		})
}

func TestGetWatchNamespace(t *testing.T) {
	Global = &Framework{
		NamespacedManPath: &fakeNamespacedManPath,
		KubeClient:        fake.NewSimpleClientset(),
	}

	t.Run("should return Watch Namespace (==Operator Namespace) WatchNamespace Env is not set",
		func(t *testing.T) {
			Global.WatchNamespace = ""
			Global.OperatorNamespace = ""
			ctx := NewContext(t)
			watchNamespace, err := ctx.GetWatchNamespace()
			assertNoError(t, err)
			operatorNamespace, err := ctx.GetOperatorNamespace()
			assertNoError(t, err)
			if len(operatorNamespace) <= 0 {
				t.Errorf("Expected non-empty operatorNamespace")
			}
			if watchNamespace != operatorNamespace {
				t.Errorf("Expected watchNamespace: %s, got %s", operatorNamespace, watchNamespace)
			}
		})

	t.Run("should return Watch Namespace (==WatchNamespace Env) WatchNamespace Env is set",
		func(t *testing.T) {
			watchNamespace := "test-watch-namespace"
			Global.WatchNamespace = watchNamespace
			Global.OperatorNamespace = ""
			os.Setenv(WatchNamespaceEnv, watchNamespace)
			defer func() {
				Global.WatchNamespace = ""
				defer os.Unsetenv(WatchNamespaceEnv)
			}()

			ctx := NewContext(t)
			got, err := ctx.GetWatchNamespace()
			assertNoError(t, err)
			if watchNamespace != got {
				t.Errorf("Expected watchNamespace: %s, got %s", watchNamespace, got)
			}
			operatorNamespace, err := ctx.GetOperatorNamespace()
			assertNoError(t, err)
			if len(operatorNamespace) <= 0 {
				t.Errorf("Expected non-empty operatorNamespace")
			}
			if watchNamespace == operatorNamespace {
				t.Errorf("Expected operator-Namespace: %v, to be different than watch-Namespace: %v",
					operatorNamespace, watchNamespace)
			}
		})

	t.Run("should return Watch Namespace (==WatchNamespace Env) even when WatchNamespace Env is set to \"\"",
		func(t *testing.T) {
			watchNamespace := ""
			Global.WatchNamespace = watchNamespace
			Global.OperatorNamespace = ""
			os.Setenv(WatchNamespaceEnv, watchNamespace)
			defer func() {
				Global.WatchNamespace = ""
				defer os.Unsetenv(WatchNamespaceEnv)
			}()

			ctx := NewContext(t)

			got, err := ctx.GetWatchNamespace()
			assertNoError(t, err)
			if watchNamespace != got {
				t.Errorf("Expected watchNamespace: %s, got %s", watchNamespace, got)
			}
			operatorNamespace, err := ctx.GetOperatorNamespace()
			assertNoError(t, err)
			if len(operatorNamespace) <= 0 {
				t.Errorf("Expected non-empty operatorNamespace")
			}
			if watchNamespace == operatorNamespace {
				t.Errorf("Expected operator-Namespace: %v, to be different than watch-Namespace: %v",
					operatorNamespace, watchNamespace)
			}
		})

	t.Run("should return Watch Namespace and Operator Namespace when both Env are set through Env var",
		func(t *testing.T) {
			watchNamespace := "test-watch-namespace"
			operatorNamespace := "test-operator-namespace"
			Global.WatchNamespace = watchNamespace
			Global.OperatorNamespace = operatorNamespace
			os.Setenv(OperatorNamespaceEnv, operatorNamespace)
			os.Setenv(WatchNamespaceEnv, watchNamespace)
			defer func() {
				Global.WatchNamespace = ""
				Global.OperatorNamespace = ""
				defer os.Unsetenv(OperatorNamespaceEnv)
				defer os.Unsetenv(WatchNamespaceEnv)
			}()

			ctx := NewContext(t)
			gotWatchNamespace, err := ctx.GetWatchNamespace()
			assertNoError(t, err)
			if watchNamespace != gotWatchNamespace {
				t.Errorf("Expected watchNamespace: %s, got %s", watchNamespace, gotWatchNamespace)
			}

			gotOperatorNamespace, err := ctx.GetOperatorNamespace()
			assertNoError(t, err)
			if len(gotOperatorNamespace) <= 0 {
				t.Errorf("Expected non-empty operatorNamespace")
			}
			if gotWatchNamespace == gotOperatorNamespace {
				t.Errorf("Expected operator-Namespace: %v, to be different than watch-Namespace: %v",
					operatorNamespace, watchNamespace)
			}
			if gotOperatorNamespace != operatorNamespace {
				t.Errorf("Expected operatorNamespace: %s, got %s", operatorNamespace, gotOperatorNamespace)
			}
		})
}

func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected no error, %v", err)
	}
}
