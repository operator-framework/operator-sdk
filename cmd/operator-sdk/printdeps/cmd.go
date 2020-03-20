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

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/internal/scaffold/helm"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/spf13/cobra"

	log "github.com/sirupsen/logrus"
)

func NewCmd() *cobra.Command {
	printDepsCmd := &cobra.Command{
		Use:   "print-deps",
		Short: "Print Golang packages and versions required to run the operator",
		Long: `The operator-sdk print-deps command prints all Golang packages and versions expected
by this version of the Operator SDK. Versions for these packages should match
those in an operator's go.mod file.
`,
		RunE: printDepsFunc,
	}
	return printDepsCmd
}

func printDepsFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
	}
	projutil.MustInProjectRoot()

	if err := printDeps(); err != nil {
		log.Fatalf("Print deps failed: %v", err)
	}
	return nil
}

func printDeps() (err error) {
	// Migrated Ansible and Helm projects will be of type OperatorTypeGo but
	// their deps files will differ from a vanilla Go project.
	switch {
	case projutil.IsOperatorAnsible():
		return ansible.PrintGoMod()
	case projutil.IsOperatorHelm():
		return helm.PrintGoMod()
	case projutil.IsOperatorGo():
		return scaffold.PrintGoMod()
	}

	return projutil.ErrUnknownOperatorType{}
}
