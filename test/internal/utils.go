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
	"os/exec"

	. "github.com/onsi/ginkgo" //nolint:golint

	kbtestutils "sigs.k8s.io/kubebuilder/test/e2e/utils"
)

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
