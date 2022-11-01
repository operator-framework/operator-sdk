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

package cli

import (
	"fmt"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ver "github.com/operator-framework/operator-sdk/internal/version"
)

var _ = Describe("printVersion", func() {
	It("prints the correct version info", func() {
		expVersion := makeVersionString()
		version := ver.GitVersion
		if version == "unknown" {
			version = ver.Version
		}
		Expect(expVersion).To(ContainSubstring(fmt.Sprintf("version: %q", version)))
		Expect(expVersion).To(ContainSubstring(fmt.Sprintf("commit: %q", ver.GitCommit)))
		Expect(expVersion).To(ContainSubstring(fmt.Sprintf("kubernetes version: %q", ver.KubernetesVersion)))
		Expect(expVersion).To(ContainSubstring(fmt.Sprintf("go version: %q", runtime.Version())))
		Expect(expVersion).To(ContainSubstring(fmt.Sprintf("GOOS: %q", runtime.GOOS)))
		Expect(expVersion).To(ContainSubstring(fmt.Sprintf("GOARCH: %q", runtime.GOARCH)))
	})
})
