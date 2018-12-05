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

package projutil

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/helm"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	SrcDir          = "src"
	mainFile        = "./cmd/manager/main.go"
	buildDockerfile = "./build/Dockerfile"
)

const (
	GopathEnv = "GOPATH"
)

// OperatorType - the type of operator
type OperatorType = string

const (
	// OperatorTypeGo - golang type of operator.
	OperatorTypeGo OperatorType = "go"
	// OperatorTypeAnsible - ansible type of operator.
	OperatorTypeAnsible OperatorType = "ansible"
	// OperatorTypeHelm - helm type of operator.
	OperatorTypeHelm OperatorType = "helm"
	// OperatorTypeUnknown - unknown type of operator.
	OperatorTypeUnknown OperatorType = "unknown"
)

// MustInProjectRoot checks if the current dir is the project root and returns the current repo's import path
// e.g github.com/example-inc/app-operator
func MustInProjectRoot() {
	// if the current directory has the "./build/dockerfile" file, then it is safe to say
	// we are at the project root.
	_, err := os.Stat(buildDockerfile)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatal("must run command in project root dir: project structure requires ./build/Dockerfile")
		}
		log.Fatalf("error: (%v) while checking if current directory is the project root", err)
	}
}

func MustGoProjectCmd(cmd *cobra.Command) {
	t := GetOperatorType()
	switch t {
	case OperatorTypeGo:
	default:
		log.Fatalf("'%s' can only be run for Go operators; %s does not exist.", cmd.CommandPath(), mainFile)
	}
}

func MustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to get working directory: (%v)", err)
	}
	return wd
}

// CheckAndGetProjectGoPkg checks if this project's repository path is rooted under $GOPATH and returns the current directory's import path
// e.g: "github.com/example-inc/app-operator"
func CheckAndGetProjectGoPkg() string {
	gopath := SetGopath(GetGopath())
	goSrc := filepath.Join(gopath, SrcDir)
	wd := MustGetwd()
	currPkg := strings.Replace(wd, goSrc+string(filepath.Separator), "", 1)
	// strip any "/" prefix from the repo path.
	return strings.TrimPrefix(currPkg, string(filepath.Separator))
}

// GetOperatorType returns type of operator is in cwd
// This function should be called after verifying the user is in project root
// e.g: "go", "ansible"
func GetOperatorType() OperatorType {
	// Assuming that if main.go exists then this is a Go operator
	if _, err := os.Stat(mainFile); err == nil {
		return OperatorTypeGo
	}
	if stat, err := os.Stat(ansible.RolesDir); err == nil && stat.IsDir() {
		return OperatorTypeAnsible
	}
	if stat, err := os.Stat(helm.HelmChartsDir); err == nil && stat.IsDir() {
		return OperatorTypeHelm
	}
	return OperatorTypeUnknown
}

// GetGopath gets GOPATH and makes sure it is set and non-empty.
func GetGopath() string {
	gopath, ok := os.LookupEnv(GopathEnv)
	if !ok || len(gopath) == 0 {
		log.Fatal("GOPATH env not set")
	}
	return gopath
}

// SetGopath sets GOPATH=currentGopath after processing a path list,
// if any, then returns the set path.
func SetGopath(currentGopath string) string {
	var newGopath string
	cwdInGopath := false
	wd := MustGetwd()
	for _, newGopath = range strings.Split(currentGopath, ":") {
		if strings.HasPrefix(filepath.Dir(wd), newGopath) {
			cwdInGopath = true
			break
		}
	}
	if !cwdInGopath {
		log.Fatalf("project not in $GOPATH")
		return ""
	}
	if err := os.Setenv(GopathEnv, newGopath); err != nil {
		log.Fatal(err)
		return ""
	}
	return newGopath
}

// CombineManifests combines a given manifest with a base manifest and adds yaml
// style separation. Nothing is appended if the manifest is empty.
func CombineManifests(base, manifest []byte) []byte {
	if len(manifest) > 0 {
		base = append(base, manifest...)
		return append(base, []byte("\n---\n")...)
	}
	return base
}

// GenerateCombinedNamespacedManifest creates a temporary manifest yaml
// containing all standard namespaced resource manifests combined into 1 file
func GenerateCombinedNamespacedManifest() (*os.File, error) {
	file, err := ioutil.TempFile("", "namespaced-manifest.yaml")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	sa, err := ioutil.ReadFile(filepath.Join(scaffold.DeployDir, scaffold.ServiceAccountYamlFile))
	if err != nil {
		log.Warnf("could not find the serviceaccount manifest: (%v)", err)
	}
	role, err := ioutil.ReadFile(filepath.Join(scaffold.DeployDir, scaffold.RoleYamlFile))
	if err != nil {
		log.Warnf("could not find role manifest: (%v)", err)
	}
	roleBinding, err := ioutil.ReadFile(filepath.Join(scaffold.DeployDir, scaffold.RoleBindingYamlFile))
	if err != nil {
		log.Warnf("could not find role_binding manifest: (%v)", err)
	}
	operator, err := ioutil.ReadFile(filepath.Join(scaffold.DeployDir, scaffold.OperatorYamlFile))
	if err != nil {
		return nil, fmt.Errorf("could not find operator manifest: (%v)", err)
	}
	combined := []byte{}
	combined = CombineManifests(combined, sa)
	combined = CombineManifests(combined, role)
	combined = CombineManifests(combined, roleBinding)
	combined = append(combined, operator...)

	if err := file.Chmod(os.FileMode(fileutil.DefaultFileMode)); err != nil {
		return nil, fmt.Errorf("could not chown temporary namespaced manifest file: (%v)", err)
	}
	if _, err := file.Write(combined); err != nil {
		return nil, fmt.Errorf("could not create temporary namespaced manifest file: (%v)", err)
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return file, nil
}

// GenerateCombinedGlobalManifest creates a temporary manifest yaml
// containing all standard global resource manifests combined into 1 file
func GenerateCombinedGlobalManifest() (*os.File, error) {
	file, err := ioutil.TempFile("", "global-manifest.yaml")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	files, err := ioutil.ReadDir(scaffold.CrdsDir)
	if err != nil {
		return nil, fmt.Errorf("could not read deploy directory: (%v)", err)
	}
	combined := []byte{}
	for _, file := range files {
		if strings.HasSuffix(file.Name(), "crd.yaml") {
			fileBytes, err := ioutil.ReadFile(filepath.Join(scaffold.CrdsDir, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("could not read file %s: (%v)", filepath.Join(scaffold.CrdsDir, file.Name()), err)
			}
			combined = CombineManifests(combined, fileBytes)
		}
	}

	if err := file.Chmod(os.FileMode(fileutil.DefaultFileMode)); err != nil {
		return nil, fmt.Errorf("could not chown temporary global manifest file: (%v)", err)
	}
	if _, err := file.Write(combined); err != nil {
		return nil, fmt.Errorf("could not create temporary global manifest file: (%v)", err)
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return file, nil
}
