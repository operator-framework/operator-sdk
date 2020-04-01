// Copyright 2020 The Operator-SDK Authors
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

package olmcatalog

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/ghodss/yaml"
	appsv1 "k8s.io/api/apps/v1"
)

func TestSetAndCheckOLMNamespaces(t *testing.T) {
	cleanupFunc := chDirWithCleanup(t, testGoDataDir)
	defer cleanupFunc()

	depBytes, err := ioutil.ReadFile(filepath.Join(scaffold.DeployDir, "operator.yaml"))
	if err != nil {
		t.Fatalf("Failed to read Deployment bytes: %v", err)
	}

	// The test operator.yaml doesn't have "olm.targetNamespaces", so first
	// check that depHasOLMNamespaces() returns false.
	dep := appsv1.Deployment{}
	if err := yaml.Unmarshal(depBytes, &dep); err != nil {
		t.Fatalf("Failed to unmarshal Deployment bytes: %v", err)
	}
	if depHasOLMNamespaces(dep) {
		t.Error("Expected depHasOLMNamespaces to return false, got true")
	}

	// Insert "olm.targetNamespaces" into WATCH_NAMESPACE and check that
	// depHasOLMNamespaces() returns true.
	setWatchNamespacesEnv(&dep)
	if !depHasOLMNamespaces(dep) {
		t.Error("Expected depHasOLMNamespaces to return true, got false")
	}

	// Overwrite WATCH_NAMESPACE and check that depHasOLMNamespaces() returns
	// false.
	overwriteContainerEnvVar(&dep, k8sutil.WatchNamespaceEnvVar, newEnvVar("FOO", "bar"))
	if depHasOLMNamespaces(dep) {
		t.Error("Expected depHasOLMNamespaces to return false, got true")
	}

	// Insert "olm.targetNamespaces" elsewhere in the deployment pod spec
	// and check that depHasOLMNamespaces() returns true.
	dep = appsv1.Deployment{}
	if err := yaml.Unmarshal(depBytes, &dep); err != nil {
		t.Fatalf("Failed to unmarshal Deployment bytes: %v", err)
	}
	dep.Spec.Template.ObjectMeta.Labels["namespace"] = olmTNMeta
	if !depHasOLMNamespaces(dep) {
		t.Error("Expected depHasOLMNamespaces to return true, got false")
	}
}
