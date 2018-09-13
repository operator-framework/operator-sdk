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

package cmdtest

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	cmdError "github.com/operator-framework/operator-sdk/commands/operator-sdk/error"
	"github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/spf13/cobra"
)

var (
	kubeconfig             string
	globalManifestPath     string
	namespacedManifestPath string
	goTestFlags            string
)

func NewTestLocalCmd() *cobra.Command {
	testCmd := &cobra.Command{
		Use:   "local <path to tests directory> [flags]",
		Short: "Run End-To-End tests locally",
		Run:   testLocalFunc,
	}
	defaultKubeConfig := ""
	homedir, ok := os.LookupEnv("HOME")
	if ok {
		defaultKubeConfig = homedir + "/.kube/config"
	}
	testCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", defaultKubeConfig, "Kubeconfig path")
	testCmd.Flags().StringVarP(&globalManifestPath, "global-init", "g", "deploy/crd.yaml", "Path to manifest for Global resources (e.g. CRD manifest)")
	testCmd.Flags().StringVarP(&namespacedManifestPath, "namespaced-init", "n", "", "Path to manifest for per-test, namespaced resources (e.g. RBAC and Operator manifest)")
	testCmd.Flags().StringVarP(&goTestFlags, "go-test-flags", "f", "", "Additional flags to pass to go test")

	return testCmd
}

func testLocalFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, fmt.Errorf("operator-sdk test local requires exactly 1 argument"))
	}
	// if no namespaced manifest path is given, combine deploy/rbac.yaml and deploy/operator.yaml
	if namespacedManifestPath == "" {
		os.Mkdir("deploy/test", os.FileMode(int(0775)))
		namespacedManifestPath = "deploy/test/namespace-manifests.yaml"
		rbac, err := ioutil.ReadFile("deploy/rbac.yaml")
		if err != nil {
			log.Fatalf("could not find rbac manifest: %v", err)
		}
		operator, err := ioutil.ReadFile("deploy/operator.yaml")
		if err != nil {
			log.Fatalf("could not find operator manifest: %v", err)
		}
		combined := append(rbac, []byte("\n---\n")...)
		combined = append(combined, operator...)
		err = ioutil.WriteFile(namespacedManifestPath, combined, os.FileMode(int(0664)))
		if err != nil {
			log.Fatalf("could not create temporary namespaced manifest file: %v", err)
		}
		defer func() {
			err := os.Remove(namespacedManifestPath)
			if err != nil {
				log.Fatalf("could not delete temporary namespace manifest file")
			}
		}()
	}
	testArgs := []string{"test", args[0] + "/..."}
	testArgs = append(testArgs, "-"+test.KubeConfigFlag, kubeconfig)
	testArgs = append(testArgs, "-"+test.NamespacedManPathFlag, namespacedManifestPath)
	testArgs = append(testArgs, "-"+test.GlobalManPathFlag, globalManifestPath)
	testArgs = append(testArgs, "-"+test.ProjRootFlag, mustGetwd())
	testArgs = append(testArgs, strings.Split(goTestFlags, " ")...)
	dc := exec.Command("go", testArgs...)
	dc.Dir = mustGetwd()
	dc.Stdout = os.Stdout
	dc.Stderr = os.Stderr
	err := dc.Run()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to exec %v %#v: %v", cmd, args, err))
	}
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to determine the full path of the current directory: %v", err))
	}
	return wd
}
