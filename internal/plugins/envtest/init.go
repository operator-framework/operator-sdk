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

// TODO: This implementation is already done for v3+. Also, it might be
// addressed on v2 as well. More info: https://github.com/kubernetes-sigs/kubebuilder/pull/1711
package envtest

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/config"
)

// controllerRuntimeVersion version to be used to download the envtest setup script
const controllerRuntimeVersion = "v0.6.3"

// RunInit modifies the project scaffolded by kubebuilder's Init plugin.
func RunInit(cfg *config.Config) error {
	// Only run these if project version is v3.
	if !cfg.IsV3() {
		return nil
	}

	// Update the scaffolded Makefile with operator-sdk recipes.
	if err := initUpdateMakefile("Makefile"); err != nil {
		return fmt.Errorf("error updating Makefile: %v", err)
	}
	return nil
}

// initUpdateMakefile updates a vanilla kubebuilder Makefile with operator-sdk recipes.
func initUpdateMakefile(filePath string) error {
	makefileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	makefileBytes = []byte(strings.Replace(string(makefileBytes),
		"# Run tests\ntest: generate fmt vet manifests\n\tgo test ./... -coverprofile cover.out",
		fmt.Sprintf(makefileTestTarget, controllerRuntimeVersion), 1))

	var mode os.FileMode = 0644
	if info, err := os.Stat(filePath); err != nil {
		mode = info.Mode()
	}
	return ioutil.WriteFile(filePath, makefileBytes, mode)
}

const makefileTestTarget = `# Run tests
ENVTEST_ASSETS_DIR = $(shell pwd)/testbin
test: generate fmt vet manifests
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/%s/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out`
