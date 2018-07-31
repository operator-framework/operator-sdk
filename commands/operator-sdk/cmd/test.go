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
	"fmt"
	"os"
	"os/exec"

	cmdError "github.com/operator-framework/operator-sdk/commands/operator-sdk/error"
	"github.com/spf13/cobra"
)

var (
	kubeconfig string
	imageName  string
	verbose    bool
)

func NewTestCmd() *cobra.Command {
	testCmd := &cobra.Command{
		Use:   "test --kubeconfig <path to kubeconfig> --image <name of operator image>",
		Short: "Run End-To-End tests",
		Run:   testFunc,
	}
	defaultKubeConfig := ""
	homedir, ok := os.LookupEnv("HOME")
	if ok {
		defaultKubeConfig = homedir + "/.kube/config"
	}
	testCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", defaultKubeConfig, "Kubeconfig path (e.g. $HOME/.kube/config)")
	testCmd.Flags().StringVarP(&imageName, "image", "i", "", "Name of image (e.g. quay.io/example-inc/test-operator:v0.0.1)")
	testCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose go test")

	return testCmd
}

func testFunc(cmd *cobra.Command, args []string) {
	testArgs := []string{"test", "./test/e2e/..."}
	if verbose {
		testArgs = append(testArgs, "-v")
	}
	dc := exec.Command("go", testArgs...)
	dc.Stdout = os.Stdout
	dc.Stderr = os.Stderr
	dc.Env = append(os.Environ(), fmt.Sprintf("%v=%v", "TEST_KUBECONFIG", kubeconfig), fmt.Sprintf("%v=%v", "TEST_IMAGE", imageName))
	err := dc.Run()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("test failed: %v", err))
	}
}
