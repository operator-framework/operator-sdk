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

var asFile bool

func NewCmd() *cobra.Command {
	printDepsCmd := &cobra.Command{
		Use:   "print-deps",
		Short: "Print Golang packages and versions required to run the operator",
		Long: `The operator-sdk print-deps command prints all Golang packages and versions expected
by this version of the Operator SDK. Versions for these packages should match
those in an operators' go.mod or Gopkg.toml file, depending on the dependency
manager chosen when initializing or migrating a project.

print-deps prints in columnar format by default. Use the --as-file flag to
print in go.mod or Gopkg.toml file format.
`,
		RunE: printDepsFunc,
	}

	printDepsCmd.Flags().BoolVar(&asFile, "as-file", false, "Print dependencies in go.mod or Gopkg.toml file format, depending on the dependency manager chosen when initializing or migrating a project")

	return printDepsCmd
}

func printDepsFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
	}

	if err := printDeps(asFile); err != nil {
		return fmt.Errorf("print deps failed: (%v)", err)
	}
	return nil
}

func printDeps(asFile bool) error {
	// Make sure the project has a dep manager file.
	mt, err := projutil.GetDepManagerType()
	if err != nil {
		return err
	}
	isDep := mt == projutil.DepManagerDep

	// Migrated Ansible and Helm projects will be of type OperatorTypeGo but
	// their deps files will differ from a vanilla Go project.
	switch {
	case projutil.IsOperatorAnsible():
		if isDep {
			return ansible.PrintDepGopkgTOML(asFile)
		}
		return ansible.PrintGoMod(asFile)
	case projutil.IsOperatorHelm():
		if isDep {
			return helm.PrintDepGopkgTOML(asFile)
		}
		return helm.PrintGoMod(asFile)
	case projutil.IsOperatorGo():
		if isDep {
			return scaffold.PrintDepGopkgTOML(asFile)
		}
		return scaffold.PrintGoMod(asFile)
	}

	return projutil.ErrUnknownOperatorType{}
}
