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

const (
	// k8sVerTag is the k8s.io/code-generator tag used to build codegen binaries.
	k8sVerTag = "kubernetes-1.11.2"
	// codegenRepo is the git repo path for k8s.io/code-generator.
	codegenRepo = "https://github.com/kubernetes/code-generator.git"
)

// K8sCodegen performs deepcopy code-generation for all custom resources under pkg/apis
func K8sCodegen() {
	projutil.MustInProjectRoot()

	repoPkg := projutil.CheckAndGetProjectGoPkg()
	codegenPkg := filepath.Join("k8s.io", "code-generator")

	codegenAbs := getCodegenRepo(codegenPkg, codegenRepo, k8sVerTag)
	installCodegenBinaries(codegenAbs)

	gvMap, err := parseGroupVersions()
	if err != nil {
		log.Fatalf("failed to parse group versions: (%v)", err)
	}
	gvStr := ""
	for g, vs := range gvMap {
		gvStr += fmt.Sprintf("%s:%v, ", g, vs)
	}

	log.Infof("Running code-generation for Custom Resource group versions: [%v]\n", gvStr)

	deepcopyGen(repoPkg, gvMap)

	log.Info("Code-generation complete.")
}

func getCodegenRepo(pkg, repo, tag string) string {
	// Using vendored codegen source files is preferred over cloning.
	codegenVendor := filepath.Join("vendor", pkg)
	if _, err := os.Stat(codegenVendor); err == nil || os.IsExist(err) {
		return codegenVendor
	}

	codegenAbs := filepath.Join(projutil.GetGopath(), "src", pkg)
	if err := os.RemoveAll(codegenAbs); err != nil {
		log.Fatal(err)
	}
	cloneCmd := exec.Command("git", "clone", "-q", repo, codegenAbs)
	if err := projutil.ExecCmd(cloneCmd); err != nil {
		log.Fatal(err)
	}
	checkoutCmd := exec.Command("git", "checkout", "-q", tag)
	checkoutCmd.Dir = codegenAbs
	if err := projutil.ExecCmd(checkoutCmd); err != nil {
		log.Fatal(err)
	}
	return codegenAbs
}

func installCodegenBinaries(abs string) {
	args := []string{
		"install",
		"./cmd/defaulter-gen",
		"./cmd/client-gen",
		"./cmd/lister-gen",
		"./cmd/informer-gen",
		"./cmd/deepcopy-gen",
	}
	if gf, ok := os.LookupEnv("GOFLAGS"); ok && len(gf) != 0 {
		sf := strings.Split(gf, " ")
		args = append(append(args[:1], sf...), args[len(sf)+1:]...)
	}
	installCmd := exec.Command("go", args...)
	installCmd.Dir = abs
	if err := projutil.ExecCmd(installCmd); err != nil {
		log.Fatal(err)
	}
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

func deepcopyGen(repoPkg string, gvMap map[string][]string) {
	apisPkg := filepath.Join(repoPkg, scaffold.ApisDir)
	args := []string{
		"--input-dirs", createFQApis(apisPkg, gvMap),
		"-O", "zz_generated.deepcopy",
		"--bounding-dirs", apisPkg,
	}
	cgPath := filepath.Join(projutil.GetGopath(), "bin", "deepcopy-gen")
	err := projutil.ExecCmd(exec.Command(cgPath, args...))
	if err != nil {
		log.Fatalf("failed to perform code-generation: %v", err)
	}
}

// createFQApis return a string of all fully qualified pkg + groups + versions
// of pkg and gvs in the format:
// "pkg/groupA/v1,pkg/groupA/v2,pkg/groupB:v1"
func createFQApis(pkg string, gvs map[string][]string) (fqStr string) {
	gn := 0
	for g, vs := range gvs {
		for vn, v := range vs {
			fqStr += filepath.Join(pkg, g, v)
			if vn < len(vs)-1 {
				fqStr += ","
			}
		}
		if gn < len(gvs)-1 {
			fqStr += ","
		}
		gn++
	}
	return fqStr
}
