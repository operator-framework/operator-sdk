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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/helm"

	docker "docker.io/go-docker"
	"github.com/coreos/go-semver/semver"
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

func IsGoOperator() bool {
	return GetOperatorType() == OperatorTypeGo
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

// IsDockerMultistage checks whether the docker daemon is version >= 17.05.0.
// Both ARG before FROM and multistage builds exist in docker version >= 17.05.0.
func IsDockerMultistage() bool {
	client, err := docker.NewEnvClient()
	if err != nil {
		log.Fatalf("get docker client: (%v)", err)
	}
	ver, err := client.ServerVersion(context.TODO())
	if err != nil {
		log.Fatalf("get docker version: (%v)", err)
	}
	dockerVer := semver.New(ver.Version)
	reqVer := semver.New("17.05.0")
	return !dockerVer.LessThan(*reqVer)
}

// IsDockerfileMultistage checks if dockerfile uses a multistage pipeline.
func IsDockerfileMultistage(dockerfile string) bool {
	df, err := ioutil.ReadFile(dockerfile)
	if err != nil {
		log.Fatal("Dockerfile not found")
	}
	return bytes.Count(df, []byte("FROM")) > 1
}

func ExecCmd(cmd *exec.Cmd) error {
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to exec %s %#v: %v", cmd.Path, cmd.Args, err)
	}
	return nil
}

func DockerBuild(dockerfile, image string, buildArgs ...string) error {
	args := []string{"build", ".", "-f", dockerfile, "-t", image}
	for _, arg := range buildArgs {
		args = append(args, "--build-arg", arg)
	}
	bcmd := exec.Command("docker", args...)
	err := ExecCmd(bcmd)
	if err != nil {
		return fmt.Errorf("error building docker image %s: %v", image, err)
	}
	return nil
}
