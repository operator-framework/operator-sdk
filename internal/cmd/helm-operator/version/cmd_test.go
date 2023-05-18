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

package version

import (
	"fmt"
	"io"
	"os"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	ver "github.com/operator-framework/operator-sdk/internal/version"
)

var _ = Describe("Running a version command", func() {
	Describe("NewCmd", func() {
		It("builds a cobra command", func() {
			cmd := NewCmd()
			Expect(cmd).NotTo(BeNil())
			Expect(cmd.Use).NotTo(Equal(""))
			Expect(cmd.Short).NotTo(Equal(""))
		})
	})
	Describe("run", func() {
		It("prints the correct version info", func() {
			r, w, _ := os.Pipe()
			tmp := os.Stdout
			defer func() {
				os.Stdout = tmp
			}()
			os.Stdout = w
			go func() {
				run()
				w.Close()
			}()
			stdout, err := io.ReadAll(r)
			Expect(err).ToNot(HaveOccurred())
			stdoutString := string(stdout)
			version := ver.GitVersion
			if version == "unknown" {
				version = ver.Version
			}
			Expect(stdoutString).To(ContainSubstring(fmt.Sprintf("version: %q", version)))
			Expect(stdoutString).To(ContainSubstring(fmt.Sprintf("commit: %q", ver.GitCommit)))
			Expect(stdoutString).To(ContainSubstring(fmt.Sprintf("kubernetes version: %q", ver.KubernetesVersion)))
			Expect(stdoutString).To(ContainSubstring(fmt.Sprintf("go version: %q", runtime.Version())))
			Expect(stdoutString).To(ContainSubstring(fmt.Sprintf("GOOS: %q", runtime.GOOS)))
			Expect(stdoutString).To(ContainSubstring(fmt.Sprintf("GOARCH: %q", runtime.GOARCH)))
		})
	})
})
