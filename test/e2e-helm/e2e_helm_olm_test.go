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

package e2e_helm_test

import (
	"path/filepath"

	. "github.com/onsi/ginkgo" //nolint:golint

	testutils "github.com/operator-framework/operator-sdk/test/internal"
)

var _ = PDescribe("Integrating Helm Projects with OLM", func() {
	Context("with operator-sdk", func() {
		BeforeEach(func() {
			By("Turning off interactive prompts for all generation tasks.")
			replace := "operator-sdk generate kustomize manifests"
			testutils.ReplaceInFile(filepath.Join(tc.Dir, "Makefile"), replace, replace+" --interactive=false")
		})

		AfterEach(func() {
		})

		It("Should allow generate the OLM bundle and run it", func() {
		})
	})
})
