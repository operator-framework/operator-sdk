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

package migrate

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	"github.com/spf13/cobra"
)

// NewCmd returns a command that will add source code to an existing non-go operator
func NewCmd() *cobra.Command {
	c := &scaffold.MigrateCmd{}

	newCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Adds source code to an operator",
		Long:  `operator-sdk migrate adds a main.go source file and any associated source files for an operator that is not of the "go" type.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
			}
			projutil.MustInProjectRoot()
			return c.Run()
		},
	}

	newCmd.Flags().StringVar(&c.DepManager, "dep-manager", "modules", `Dependency manager the new project will use (choices: "dep", "modules")`)
	newCmd.Flags().StringVar(&c.HeaderFile, "header-file", "", "Path to file containing headers for generated Go files. Copied to hack/boilerplate.go.txt")

	return newCmd
}
