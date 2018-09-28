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

type testLocalConfig struct {
	kubeconfig        string
	globalManPath     string
	namespacedManPath string
	goTestFlags       string
	namespace         string
}

var tlConfig testLocalConfig

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
	testCmd.Flags().StringVar(&tlConfig.kubeconfig, "kubeconfig", defaultKubeConfig, "Kubeconfig path")
	testCmd.Flags().StringVar(&tlConfig.globalManPath, "global-manifest", "deploy/crd.yaml", "Path to manifest for Global resources (e.g. CRD manifest)")
	testCmd.Flags().StringVar(&tlConfig.namespacedManPath, "namespaced-manifest", "", "Path to manifest for per-test, namespaced resources (e.g. RBAC and Operator manifest)")
	testCmd.Flags().StringVar(&tlConfig.goTestFlags, "go-test-flags", "", "Additional flags to pass to go test")
	testCmd.Flags().StringVar(&tlConfig.namespace, "namespace", "", "If non-empty, single namespace to run tests in")

	return testCmd
}

func testLocalFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, fmt.Errorf("operator-sdk test local requires exactly 1 argument"))
	}
	// if no namespaced manifest path is given, combine deploy/sa.yaml, deploy/rbac.yaml and deploy/operator.yaml
	if tlConfig.namespacedManPath == "" {
		os.Mkdir("deploy/test", os.FileMode(int(0775)))
		tlConfig.namespacedManPath = "deploy/test/namespace-manifests.yaml"
		sa, err := ioutil.ReadFile("deploy/sa.yaml")
		if err != nil {
			log.Fatalf("could not find sa manifest: %v", err)
		}
		rbac, err := ioutil.ReadFile("deploy/rbac.yaml")
		if err != nil {
			log.Fatalf("could not find rbac manifest: %v", err)
		}
		operator, err := ioutil.ReadFile("deploy/operator.yaml")
		if err != nil {
			log.Fatalf("could not find operator manifest: %v", err)
		}
		combined := append(sa, []byte("\n---\n")...)
		combined = append(combined, rbac...)
		combined = append(combined, []byte("\n---\n")...)
		combined = append(combined, operator...)
		err = ioutil.WriteFile(tlConfig.namespacedManPath, combined, os.FileMode(int(0664)))
		if err != nil {
			log.Fatalf("could not create temporary namespaced manifest file: %v", err)
		}
		defer func() {
			err := os.Remove(tlConfig.namespacedManPath)
			if err != nil {
				log.Fatalf("could not delete temporary namespace manifest file")
			}
		}()
	}
	testArgs := []string{"test", args[0] + "/..."}
	testArgs = append(testArgs, "-"+test.KubeConfigFlag, tlConfig.kubeconfig)
	testArgs = append(testArgs, "-"+test.NamespacedManPathFlag, tlConfig.namespacedManPath)
	testArgs = append(testArgs, "-"+test.GlobalManPathFlag, tlConfig.globalManPath)
	testArgs = append(testArgs, "-"+test.ProjRootFlag, mustGetwd())
	// if we do the append using an empty go flags, it inserts an empty arg, which causes
	// any later flags to be ignored
	if tlConfig.goTestFlags != "" {
		testArgs = append(testArgs, strings.Split(tlConfig.goTestFlags, " ")...)
	}
	if tlConfig.namespace != "" {
		testArgs = append(testArgs, "-"+test.SingleNamespaceFlag, "-parallel=1")
	}
	dc := exec.Command("go", testArgs...)
	dc.Env = append(os.Environ(), fmt.Sprintf("%v=%v", test.TestNamespaceEnv, tlConfig.namespace))
	dc.Dir = mustGetwd()
	dc.Stdout = os.Stdout
	dc.Stderr = os.Stderr
	err := dc.Run()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to exec `go %s`: %v", strings.Join(testArgs, " "), err))
	}
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to determine the full path of the current directory: %v", err))
	}
	return wd
}
