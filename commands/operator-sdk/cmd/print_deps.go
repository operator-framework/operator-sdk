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

package cmd

import (
	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var asFile bool

func NewPrintDepsCmd() *cobra.Command {
	printDepsCmd := &cobra.Command{
		Use:   "print-deps",
		Short: "Print Golang packages and versions required to run the operator",
		Long: `The operator-sdk print-deps command prints all Golang packages and versions expected
by this version of the Operator SDK. Versions for these packages should match
those in an operators' Gopkg.toml file.

print-deps prints in columnar format by default. Use the --as-file flag to
print in Gopkg.toml file format.
`,
		Run: printDepsFunc,
	}

	printDepsCmd.Flags().BoolVar(&asFile, "as-file", false, "Print dependencies in Gopkg.toml file format.")

	return printDepsCmd
}

func printDepsFunc(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		log.Fatal("print-deps command does not take any arguments")
	}
	if asFile {
		scaffold.PrintDepsAsFile()
	} else {
		if err := scaffold.PrintDeps(); err != nil {
			log.Fatalf("print deps: (%v)", err)
		}
	}
}
