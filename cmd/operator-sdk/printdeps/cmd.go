// Copyright 2018 The Operator-SDK Authors
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

package printdeps

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/helm"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/spf13/cobra"
)

var depManager string

func NewCmd() *cobra.Command {
	printDepsCmd := &cobra.Command{
		Use:   "print-deps",
		Short: "Print Golang packages and versions required to run the operator",
		Long: `The operator-sdk print-deps command prints all Golang packages and versions expected
by this version of the Operator SDK. Versions for these packages should match
those in an operators' go.mod or Gopkg.toml file, depending on the dependency
manager chosen when initializing or migrating a project.
`,
		RunE: printDepsFunc,
	}

	printDepsCmd.Flags().StringVar(&depManager, "dep-manager", "", `Dependency manager file type to print (choices: "dep", "modules")`)

	return printDepsCmd
}

func printDepsFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
	}
	projutil.MustInProjectRoot()

	if err := printDeps(depManager); err != nil {
		return fmt.Errorf("print deps failed: %v", err)
	}
	return nil
}

func printDeps(depManager string) (err error) {
	// Use depManager if set. Fall back to the project's current dep manager
	// type if unset.
	mt := projutil.DepManagerType(depManager)
	if mt != "" {
		if mt != projutil.DepManagerDep && mt != projutil.DepManagerGoMod {
			return projutil.ErrInvalidDepManager(mt)
		}
	} else if mt, err = projutil.GetDepManagerType(); err != nil {
		return err
	}
	isDep := mt == projutil.DepManagerDep

	// Migrated Ansible and Helm projects will be of type OperatorTypeGo but
	// their deps files will differ from a vanilla Go project.
	switch {
	case projutil.IsOperatorAnsible():
		if isDep {
			return ansible.PrintDepGopkgTOML()
		}
		return ansible.PrintGoMod()
	case projutil.IsOperatorHelm():
		if isDep {
			return helm.PrintDepGopkgTOML()
		}
		return helm.PrintGoMod()
	case projutil.IsOperatorGo():
		if isDep {
			return scaffold.PrintDepGopkgTOML()
		}
		return scaffold.PrintGoMod()
	}

	return projutil.ErrUnknownOperatorType{}
}
