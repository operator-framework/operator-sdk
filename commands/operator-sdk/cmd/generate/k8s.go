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
	"os/exec"
	"path/filepath"
	"strings"

	genutil "github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/generate/internal"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	log "github.com/sirupsen/logrus"
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
		log.Fatalf("Command %s doesn't accept any arguments", cmd.CommandPath())
	}

	// Only Go projects can generate k8s deepcopy code.
	projutil.MustGoProjectCmd(cmd)

	K8sCodegen()
}

// K8sCodegen performs deepcopy code-generation for all custom resources under pkg/apis
func K8sCodegen() {
	projutil.MustInProjectRoot()

	wd := projutil.MustGetwd()
	repoPkg := projutil.CheckAndGetProjectGoPkg()
	srcDir := filepath.Join(wd, "vendor", "k8s.io", "code-generator")
	binDir := filepath.Join(wd, scaffold.BuildBinDir)

	buildCodegenBinaries(binDir, srcDir)

	gvMap, err := genutil.ParseGroupVersions()
	if err != nil {
		log.Fatalf("Failed to parse group versions: (%v)", err)
	}
	gvb := &strings.Builder{}
	for g, vs := range gvMap {
		gvb.WriteString(fmt.Sprintf("%s:%v, ", g, vs))
	}

	log.Infof("Running deepcopy code-generation for Custom Resource group versions: [%v]\n", gvb.String())

	deepcopyGen(binDir, repoPkg, gvMap)

	log.Info("Code-generation complete.")
}

func buildCodegenBinaries(binDir, codegenSrcDir string) {
	genDirs := []string{
		"./cmd/defaulter-gen",
		"./cmd/client-gen",
		"./cmd/lister-gen",
		"./cmd/informer-gen",
		"./cmd/deepcopy-gen",
	}
	err := genutil.BuildCodegenBinaries(genDirs, binDir, codegenSrcDir)
	if err != nil {
		log.Fatal(err)
	}
}

func deepcopyGen(binDir, repoPkg string, gvMap map[string][]string) {
	apisPkg := filepath.Join(repoPkg, scaffold.ApisDir)
	args := []string{
		"--input-dirs", genutil.CreateFQApis(apisPkg, gvMap),
		"--output-file-base", "zz_generated.deepcopy",
		"--bounding-dirs", apisPkg,
	}
	cgPath := filepath.Join(binDir, "deepcopy-gen")
	err := projutil.ExecCmd(exec.Command(cgPath, args...))
	if err != nil {
		log.Fatalf("Failed to perform code-generation: %v", err)
	}
}
