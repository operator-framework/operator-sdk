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

package test

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/spf13/cobra"
)

func newTestLocalCmd() *cobra.Command {
	c := &test.TestCmd{}
	cg := &test.LocalGoCmd{}
	ca := &test.LocalAnsibleCmd{}

	testCmd := &cobra.Command{
		Use:   "local <path to tests directory> [flags]",
		Short: "Run End-To-End tests locally",
		RunE: func(cmd *cobra.Command, args []string) error {
			switch t := projutil.GetOperatorType(); t {
			case projutil.OperatorTypeGo:
				if len(args) != 1 {
					return fmt.Errorf("command %s requires exactly one argument", cmd.CommandPath())
				}
				cg.TestPath = args[0]
				cg.TestCmd = *c
				return cg.Run()
			case projutil.OperatorTypeAnsible:
				if len(args) != 0 {
					return fmt.Errorf("command %s doesn't accept any arguments", cmd.CommandPath())
				}
				ca.TestCmd = *c
				return ca.Run()
			case projutil.OperatorTypeHelm:
				return fmt.Errorf("`test local` for Helm operators is not implemented")
			}
			return projutil.ErrUnknownOperatorType{}
		},
	}

	testCmd.Flags().StringVar(&c.KubeconfigPath, "kubeconfig", "", "Kubeconfig path")
	testCmd.Flags().StringVar(&c.Namespace, "namespace", "", "If non-empty, single namespace to run tests in")

	testCmd.Flags().StringVar(&cg.GlobalManPath, "global-manifest", "", "Path to manifest for Global resources (e.g. CRD manifests)")
	testCmd.Flags().StringVar(&cg.NamespacedManPath, "namespaced-manifest", "", "Path to manifest for per-test, namespaced resources (e.g. RBAC and Operator manifest)")
	testCmd.Flags().StringVar(&cg.GoTestFlags, "go-test-flags", "", "Additional flags to pass to go test")
	testCmd.Flags().StringVar(&cg.Image, "image", "", "Use a different operator image from the one specified in the namespaced manifest")
	testCmd.Flags().BoolVar(&cg.UpLocal, "up-local", false, "Enable running operator locally with go run instead of as an image in the cluster")
	testCmd.Flags().BoolVar(&cg.NoSetup, "no-setup", false, "Disable test resource creation")

	testCmd.Flags().StringVar(&ca.MoleculeTestFlags, "molecule-test-flags", "", "Additional flags to pass to molecule test")

	return testCmd
}
