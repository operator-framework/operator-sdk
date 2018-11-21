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

func NewPrintDepsCmd() *cobra.Command {
	printDepsCmd := &cobra.Command{
		Use:   "print-deps",
		Short: "Print dependencies expected by the Operator SDK",
		Long: `The operator-sdk print-deps command prints all dependencies expected by this
version of the Operator SDK. Versions for these dependencies should match those
in an operators' Gopkg.toml file.`,
		Run: printDepsFunc,
	}

	return printDepsCmd
}

func printDepsFunc(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		log.Fatal("print-deps command does not take any arguments")
	}
	if err := scaffold.PrintGopkgDeps(); err != nil {
		log.Fatalf("print deps: (%v)", err)
	}
}
