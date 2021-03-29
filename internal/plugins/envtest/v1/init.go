// Copyright 2021 The Operator-SDK Authors
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

package v1

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/afero"
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
)

const (
	// controllerRuntimeVersion version to be used to download the envtest setup script
	controllerRuntimeVersion = "v0.6.3"

	filePath = "Makefile"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config config.Config
}

func (s *initSubcommand) InjectConfig(c config.Config) error {
	s.config = c

	return nil
}

func (s *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	makefileBytes, err := afero.ReadFile(fs.FS, filePath)
	if err != nil {
		return err
	}

	makefileBytes = []byte(strings.Replace(string(makefileBytes), oldMakefileTestTarget, fmt.Sprintf(makefileTestTarget, controllerRuntimeVersion), 1))

	var mode os.FileMode = 0644
	if info, err := fs.FS.Stat(filePath); err == nil {
		mode = info.Mode()
	}
	if err := ioutil.WriteFile(filePath, makefileBytes, mode); err != nil {
		return fmt.Errorf("error updating Makefile: %w", err)
	}

	return nil
}

const (
	oldMakefileTestTarget = `# Run tests
test: manifests generate fmt vet
	go test ./... -coverprofile cover.out`
	makefileTestTarget = `# Run tests
ENVTEST_ASSETS_DIR = $(shell pwd)/testbin
test: manifests generate fmt vet
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/%s/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out`
)
