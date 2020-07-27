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

package run

import (
	"github.com/spf13/cobra"

	"github.com/operator-framework/operator-sdk/internal/cmd/operator-sdk/run/packagemanifests"
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run an Operator in a variety of environments",
		Long: `This command has subcommands that will deploy your Operator with OLM.
Currently only the package manifests format is supported via the 'packagemanifests' subcommand.
Run 'operator-sdk run --help' for more information.
`,
	}

	cmd.AddCommand(
		packagemanifests.NewCmd(),
	)

	return cmd
}
