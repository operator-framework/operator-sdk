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

package generate

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/cmdutil"
	cmdError "github.com/operator-framework/operator-sdk/commands/operator-sdk/error"

	"github.com/spf13/cobra"
)

const (
	k8sGenerated = "./tmp/codegen/update-generated.sh"
)

func NewGenerateK8SCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "k8s",
		Short: "Generates Kubernetes code for custom resource",
		Long: `k8s generator generates code for custom resource given the API spec
to comply with kube-API requirements.
`,
		Run: k8sFunc,
	}
}

func k8sFunc(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, errors.New("k8s command doesn't accept any arguments."))
	}
	K8sCodegen()
}

// K8sCodegen performs deepcopy code-generation for all custom resources under pkg/apis
func K8sCodegen() {
	repoPkg := cmdutil.MustInProjectRoot()
	outputPkg := filepath.Join(repoPkg, "pkg/generated")
	apisPkg := filepath.Join(repoPkg, "pkg/apis")
	groupVersions, err := parseGroupVersions()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to perform code-generation: %v", err))
	}

	fmt.Fprintf(os.Stdout, "Running code-generation for custom resource group versions: [%s]\n", groupVersions)
	// TODO: Replace generate-groups.sh by building the vendored generators(deepcopy, lister etc)
	// and running them directly
	// TODO: remove dependency on boilerplate.go.txt
	genGroupsCmd := "vendor/k8s.io/code-generator/generate-groups.sh"
	args := []string{
		"deepcopy",
		outputPkg,
		apisPkg,
		groupVersions,
		"--go-header-file", "./scripts/codegen/boilerplate.go.txt",
	}
	out, err := exec.Command(genGroupsCmd, args...).CombinedOutput()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to perform code-generation: (%v)", string(out)))
	}
	fmt.Fprintln(os.Stdout, string(out))
}

// getGroupVersions parses the layout of pkg/apis to return the API groups and versions
// in the format "groupA:v1,v2 groupB:v1 groupC:v2",
// as required by the generate-groups.sh script
func parseGroupVersions() (string, error) {
	var groupVersions string
	groups, err := ioutil.ReadDir(filepath.Join("pkg", "apis"))
	if err != nil {
		return "", fmt.Errorf("could not read pkg/apis directory to find api Versions: %v", err)
	}
	for _, g := range groups {
		// TODO: Ignore other files besides pkg/apis/group/version
		groupVersion := g.Name() + ":"
		if g.IsDir() {
			versions, err := ioutil.ReadDir(filepath.Join("pkg", "apis", g.Name()))
			if err != nil {
				return "", fmt.Errorf("could not read pkg/apis/%s directory to find api Versions: %v", g.Name(), err)
			}
			// TODO: regex check to ensure only dirs with acceptable version names are picked
			// e.g v1,v1alpha1,v1beta1 etc
			for _, v := range versions {
				if v.IsDir() {
					groupVersion = groupVersion + v.Name() + ","
				}
			}
		}
		groupVersions = groupVersions + groupVersion + " "
	}
	return groupVersions, nil
}
