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
	"os"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/spf13/cobra"
)

var (
	testLocation     string
	kubeconfig       string
	crdManifestPath  string
	opManifestPath   string
	rbacManifestPath string
	goTestFlags      string
)

func NewTestCmd() *cobra.Command {
	testCmd := &cobra.Command{
		Use:   "test --test-location <path to tests directory> [flags]",
		Short: "Run End-To-End tests",
		Run:   testFunc,
	}
	defaultKubeConfig := ""
	homedir, ok := os.LookupEnv("HOME")
	if ok {
		defaultKubeConfig = homedir + "/.kube/config"
	}
	testCmd.Flags().StringVarP(&testLocation, "test-location", "t", "", "Location of test files (e.g. ./test/e2e/)")
	testCmd.MarkFlagRequired("test-location")
	testCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", defaultKubeConfig, "Kubeconfig path")
	testCmd.Flags().StringVarP(&crdManifestPath, "crd", "c", "deploy/crd.yaml", "Path to CRD manifest")
	testCmd.Flags().StringVarP(&opManifestPath, "operator", "o", "deploy/operator.yaml", "Path to operator manifest")
	testCmd.Flags().StringVarP(&rbacManifestPath, "rbac", "r", "deploy/rbac.yaml", "Path to RBAC manifest")
	testCmd.Flags().StringVarP(&goTestFlags, "go-test-flags", "g", "", "Additional flags to pass to go test")

	return testCmd
}

func testFunc(cmd *cobra.Command, args []string) {
	testArgs := []string{"test", testLocation + "/..."}
	testArgs = append(testArgs, "-"+test.KubeConfigFlag, kubeconfig)
	testArgs = append(testArgs, "-"+test.CrdManPathFlag, crdManifestPath)
	testArgs = append(testArgs, "-"+test.OpManPathFlag, opManifestPath)
	testArgs = append(testArgs, "-"+test.RbacManPathFlag, rbacManifestPath)
	testArgs = append(testArgs, "-"+test.ProjRootFlag, mustGetwd())
	testArgs = append(testArgs, strings.Split(goTestFlags, " ")...)
	execCmd(os.Stdout, "go", testArgs...)
}
