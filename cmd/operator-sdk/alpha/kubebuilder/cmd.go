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

package kubebuilder

import (
	"fmt"
	"os/exec"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/generate"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/google/shlex"
	"github.com/spf13/cobra"
)

var kbFlags string

//nolint:lll
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kubebuilder [init/create cmds]",
		Short: "Kubebuilder aligned subcommands",
		Long: `
This subcommand is a placeholder to test the integration of Kubebuilder and Operator SDK
subcommands until a plugin system is available to integrate the Kubebuilder CLI.
See: https://github.com/kubernetes-sigs/kubebuilder/pull/1250
and https://github.com/operator-framework/operator-sdk/blob/master/doc/proposals/openshift-4.4/kubebuilder-integration.md
`,
		RunE:   runKubebuilder,
		Hidden: true,

		Example: `
	# Initialize your project
	operator-sdk alpha kubebuilder init --kb-flags="--domain example.com --license apache2 --owner \"The Kubernetes authors\""

	# Create a frigates API with Group: ship, Version: v1beta1 and Kind: Frigate
	operator-sdk alpha kubebuilder create api --kb-flags="--group ship --version v1beta1 --kind Frigate"
`,
	}

	cmd.Flags().StringVar(&kbFlags, "kb-flags", "", "Extra kubebuilder flags passed to init/create commands \"--group cache --version=v1alpha1\"")

	cmd.AddCommand(
		generate.NewCmd(),
		// newScorecardCmd(),
		// newTestFrameworkCmd(),
	)
	return cmd
}

func runKubebuilder(cmd *cobra.Command, args []string) error {
	splitArgs, err := shlex.Split(kbFlags)
	if err != nil {
		return fmt.Errorf("kb-flags is not parseable: %v", err)
	}
	args = append(args, splitArgs...)

	kbCmd := exec.Command("kubebuilder", args...)
	return projutil.ExecCmd(kbCmd)
}
