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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
		log.Fatal("k8s command doesn't accept any arguments")
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

	gvMap, err := parseGroupVersions()
	if err != nil {
		log.Fatalf("failed to parse group versions: (%v)", err)
	}
	gvb := &strings.Builder{}
	for g, vs := range gvMap {
		gvb.WriteString(fmt.Sprintf("%s:%v, ", g, vs))
	}

	log.Infof("Running code-generation for Custom Resource group versions: [%v]\n", gvb.String())

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
	for _, gd := range genDirs {
		err := runGoBuildCodegen(binDir, codegenSrcDir, gd)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func runGoBuildCodegen(binDir, repoDir, genDir string) error {
	binPath := filepath.Join(binDir, filepath.Base(genDir))
	installCmd := exec.Command("go", "build", "-o", binPath, genDir)
	installCmd.Dir = repoDir
	isVerbose := false
	if gf, ok := os.LookupEnv("GOFLAGS"); ok && len(gf) != 0 {
		installCmd.Env = append(os.Environ(), "GOFLAGS="+gf)
		if strings.Contains(gf, "-v") {
			isVerbose = true
		}
	}
	if isVerbose {
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
	} else {
		installCmd.Stdout = ioutil.Discard
		installCmd.Stderr = ioutil.Discard
	}
	return installCmd.Run()
}

// parseGroupVersions parses the layout of pkg/apis to return a map of
// API groups to versions.
func parseGroupVersions() (map[string][]string, error) {
	gvs := make(map[string][]string)
	groups, err := ioutil.ReadDir(scaffold.ApisDir)
	if err != nil {
		return nil, fmt.Errorf("could not read pkg/apis directory to find api Versions: %v", err)
	}

	for _, g := range groups {
		if g.IsDir() {
			groupDir := filepath.Join(scaffold.ApisDir, g.Name())
			versions, err := ioutil.ReadDir(groupDir)
			if err != nil {
				return nil, fmt.Errorf("could not read %s directory to find api Versions: %v", groupDir, err)
			}

			gvs[g.Name()] = make([]string, 0)
			for _, v := range versions {
				if v.IsDir() && scaffold.ResourceVersionRegexp.MatchString(v.Name()) {
					gvs[g.Name()] = append(gvs[g.Name()], v.Name())
				}
			}
		}
	}

	if len(gvs) == 0 {
		return nil, fmt.Errorf("no groups or versions found in %s", scaffold.ApisDir)
	}
	return gvs, nil
}

// createFQApis return a string of all fully qualified pkg + groups + versions
// of pkg and gvs in the format:
// "pkg/groupA/v1,pkg/groupA/v2,pkg/groupB/v1"
func createFQApis(pkg string, gvs map[string][]string) string {
	gn := 0
	sb := &strings.Builder{}
	for g, vs := range gvs {
		for vn, v := range vs {
			sb.WriteString(filepath.Join(pkg, g, v))
			if vn < len(vs)-1 {
				sb.WriteString(",")
			}
		}
		if gn < len(gvs)-1 {
			sb.WriteString(",")
		}
		gn++
	}
	return sb.String()
}

func deepcopyGen(binDir, repoPkg string, gvMap map[string][]string) {
	apisPkg := filepath.Join(repoPkg, scaffold.ApisDir)
	args := []string{
		"--input-dirs", createFQApis(apisPkg, gvMap),
		"-O", "zz_generated.deepcopy",
		"--bounding-dirs", apisPkg,
	}
	cgPath := filepath.Join(binDir, "deepcopy-gen")
	err := projutil.ExecCmd(exec.Command(cgPath, args...))
	if err != nil {
		log.Fatalf("failed to perform code-generation: %v", err)
	}
}
