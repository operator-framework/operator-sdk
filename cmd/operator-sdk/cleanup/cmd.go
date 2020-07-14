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

package cleanup

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/cleanup/packagemanifests"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up an Operator deployed with the 'run' subcommand",
		Long: `This command has subcommands that will destroy an Operator deployed with OLM.
Currently only the package manifests format is supported via the 'packagemanifests' subcommand.
Run 'operator-sdk cleanup --help' for more information.
`,
	}

	cmd.AddCommand(
		packagemanifests.NewCmd(),
	)

	return cmd
}
