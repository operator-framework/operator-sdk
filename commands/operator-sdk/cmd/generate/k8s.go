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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	"github.com/spf13/cobra"
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
		log.Fatalf("k8s command doesn't accept any arguments.")
	}
	K8sCodegen()
}

// K8sCodegen performs deepcopy code-generation for all custom resources under pkg/apis
func K8sCodegen() {
	projutil.MustInProjectRoot()
	repoPkg := projutil.CheckAndGetCurrPkg()
	outputPkg := filepath.Join(repoPkg, "pkg/generated")
	apisPkg := filepath.Join(repoPkg, scaffold.ApisDir)
	groupVersions, err := parseGroupVersions()
	if err != nil {
		log.Fatalf("failed to parse group versions: (%v)", err)
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
	}
	out, err := exec.Command(genGroupsCmd, args...).CombinedOutput()
	if err != nil {
		log.Fatalf("failed to perform code-generation: (%v)", err)
	}
	fmt.Fprintln(os.Stdout, string(out))
}

// getGroupVersions parses the layout of pkg/apis to return the API groups and versions
// in the format "groupA:v1,v2 groupB:v1 groupC:v2",
// as required by the generate-groups.sh script
func parseGroupVersions() (string, error) {
	var groupVersions string
	groups, err := ioutil.ReadDir(scaffold.ApisDir)
	if err != nil {
		return "", fmt.Errorf("could not read pkg/apis directory to find api Versions: %v", err)
	}

	for _, g := range groups {
		if g.IsDir() {
			groupDir := filepath.Join(scaffold.ApisDir, g.Name())
			versions, err := ioutil.ReadDir(groupDir)
			if err != nil {
				return "", fmt.Errorf("could not read %s directory to find api Versions: %v", groupDir, err)
			}

			groupVersion := ""
			for _, v := range versions {
				if v.IsDir() && scaffold.ResourceVersionRegexp.MatchString(v.Name()) {
					groupVersion = groupVersion + v.Name() + ","
				}
			}
			groupVersions += fmt.Sprintf("%s:%s ", g.Name(), groupVersion)
		}
	}

	return groupVersions, nil
}
