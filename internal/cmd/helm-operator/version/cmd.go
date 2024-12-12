// Copyright 2019 The Operator-SDK Authors
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
	"runtime"

	"github.com/spf13/cobra"

	ver "github.com/operator-framework/operator-sdk/internal/version"
)

func NewCmd() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Prints the version of operator-sdk",
		Run: func(_ *cobra.Command, _ []string) {
			run()
		},
	}
	return versionCmd
}

func run() {
	version := ver.GitVersion
	if version == "unknown" {
		version = ver.Version
	}
	fmt.Printf("helm-operator version: %q, commit: %q, kubernetes version: %q, go version: %q, GOOS: %q, GOARCH: %q\n",
		version, ver.GitCommit, ver.KubernetesVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
