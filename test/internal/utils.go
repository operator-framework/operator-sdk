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

package internal

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo" //nolint:golint
	. "github.com/onsi/gomega" //nolint:golint

	kbtestutils "sigs.k8s.io/kubebuilder/test/e2e/utils"
)

// Makefile fragments to add to the base Makefile just to ensure the packagemanifests feature
const makefilePackagemanifests = `
# Options for "packagemanifests".
ifneq ($(origin CHANNEL), undefined)
PKG_CHANNELS := --channel=$(CHANNEL)
endif
ifeq ($(IS_CHANNEL_DEFAULT), 1)
PKG_IS_DEFAULT_CHANNEL := --default-channel
endif
PKG_MAN_OPTS ?= $(PKG_CHANNELS) $(PKG_IS_DEFAULT_CHANNEL)

# Generate package manifests.
packagemanifests: kustomize
	operator-sdk generate kustomize manifests -q --interactive=false
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate packagemanifests -q --version $(VERSION) $(PKG_MAN_OPTS)
`

// TestContext wraps kubebuilder's e2e TestContext.
type TestContext struct {
	*kbtestutils.TestContext
}

// NewTestContext returns a TestContext containing a new kubebuilder TestContext.
func NewTestContext(env ...string) (tc TestContext, err error) {
	tc.TestContext, err = kbtestutils.NewTestContext("operator-sdk", env...)
	return tc, err
}

// InstallOLM runs 'operator-sdk olm install' and returns any errors emitted by that command.
func (tc TestContext) InstallOLM() error {
	cmd := exec.Command(tc.BinaryName, "olm", "install", "--timeout", "4m")
	_, err := tc.Run(cmd)
	return err
}

// InstallOLM runs 'operator-sdk olm uninstall' and logs any errors emitted by that command.
func (tc TestContext) UninstallOLM() {
	cmd := exec.Command(tc.BinaryName, "olm", "uninstall")
	if _, err := tc.Run(cmd); err != nil {
		fmt.Fprintln(GinkgoWriter, "warning: error when uninstalling OLM:", err)
	}
}

// KustomizeBuild runs 'kustomize build <dir>' and returns its output and an error if any.
func (tc TestContext) KustomizeBuild(dir string) ([]byte, error) {
	return tc.Run(exec.Command("kustomize", "build", dir))
}

// AddPackagemanifestsTarget will append the packagemanifests target to the makefile
// in order to test the steps described in the docs.
// More info:  https://master.sdk.operatorframework.io/docs/olm-integration/generation/#package-manifests-formats
func (tc TestContext) AddPackagemanifestsTarget() error {
	makefileBytes, err := ioutil.ReadFile(filepath.Join(tc.Dir, "Makefile"))
	if err != nil {
		return err
	}

	makefileBytes = append([]byte(makefilePackagemanifests), makefileBytes...)
	err = ioutil.WriteFile(filepath.Join(tc.Dir, "Makefile"), makefileBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}

// ReplaceInFile replaces all instances of old with new in the file at path.
func ReplaceInFile(path, old, new string) {
	info, err := os.Stat(path)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	b, err := ioutil.ReadFile(path)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	s := strings.Replace(string(b), old, new, -1)
	err = ioutil.WriteFile(path, []byte(s), info.Mode())
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
}

// LoadImageToKindCluster loads a local docker image with the name informed to the kind cluster
func (tc TestContext) LoadImageToKindClusterWithName(image string) error {
	kindOptions := []string{"load", "docker-image", image}
	cmd := exec.Command("kind", kindOptions...)
	_, err := tc.Run(cmd)
	return err
}
