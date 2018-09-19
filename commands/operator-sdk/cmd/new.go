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
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/scaffold"

	"github.com/spf13/cobra"
)

func NewNewCmd() *cobra.Command {
	newCmd := &cobra.Command{
		Use:   "new <project-name>",
		Short: "Creates a new operator application",
		Long: `The operator-sdk new command creates a new operator application and 
generates a default directory layout based on the input <project-name>. 

<project-name> is the project name of the new operator. (e.g app-operator)

For example:
	$ mkdir $GOPATH/src/github.com/example.com/
	$ cd $GOPATH/src/github.com/example.com/
	$ operator-sdk new app-operator
generates a skeletal app-operator application in $GOPATH/src/github.com/example.com/app-operator.
`,
		Run: newFunc,
	}

	newCmd.Flags().BoolVar(&skipGit, "skip-git-init", false, "Do not init the directory as a git repository")

	return newCmd
}

var (
	projectName string
	skipGit     bool
)

const (
	gopath    = "GOPATH"
	src       = "src"
	dep       = "dep"
	ensureCmd = "ensure"

	defaultDirFileMode  = 0750
	defaultFileMode     = 0644
	defaultExecFileMode = 0744
)

func newFunc(cmd *cobra.Command, args []string) {
	parse(args)
	mustBeNewProject()
	doScaffold()
	pullDep()
	initGit()
}

func parse(args []string) {
	if len(args) != 1 {
		log.Fatal("new command needs 1 argument")
	}
	projectName = args[0]
	if len(projectName) == 0 {
		log.Fatal("project-name must not be empty")
	}
}

// mustBeNewProject checks if the given project exists under the current diretory.
// it exits with error when the project exists.
func mustBeNewProject() {
	fp := filepath.Join(mustGetwd(), projectName)
	stat, err := os.Stat(fp)
	if err != nil && os.IsNotExist(err) {
		return
	}
	if err != nil {
		log.Fatalf("failed to determine if project (%v) exists", projectName)
	}
	if stat.IsDir() {
		log.Fatalf("project (%v) exists. please use a different project name or delete the existing one", projectName)
	}
}

func doScaffold() {
	// create cmd/manager dir
	fullProjectPath := filepath.Join(mustGetwd(), projectName)
	cmdDir := filepath.Join(fullProjectPath, "cmd", "manager")
	if err := os.MkdirAll(cmdDir, defaultDirFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", cmdDir, err)
	}

	// generate cmd/manager/main.go
	cmdFilePath := filepath.Join(cmdDir, "main.go")
	projectPath := repoPath()
	cmdgen := scaffold.NewCmdCodegen(&scaffold.CmdInput{ProjectPath: projectPath})
	buf := &bytes.Buffer{}
	err := cmdgen.Render(buf)
	if err != nil {
		log.Fatalf("failed to render the template for (%v): %v", cmdFilePath, err)
	}
	err = writeFileAndPrint(cmdFilePath, buf.Bytes(), defaultFileMode)
	if err != nil {
		log.Fatalf("failed to create %v: %v", cmdFilePath, err)
	}

	// create pkg/apis dir
	apisDir := filepath.Join(fullProjectPath, "pkg", "apis")
	if err := os.MkdirAll(apisDir, defaultDirFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", cmdDir, err)
	}

	// generate pkg/apis/apis.go
	apisFilePath := filepath.Join(apisDir, "apis.go")
	apisgen := scaffold.NewAPIsCodegen()
	buf = &bytes.Buffer{}
	err = apisgen.Render(buf)
	if err != nil {
		log.Fatalf("failed to render the template for (%v): %v", apisFilePath, err)
	}
	err = writeFileAndPrint(apisFilePath, buf.Bytes(), defaultFileMode)
	if err != nil {
		log.Fatalf("failed to create %v: %v", apisFilePath, err)
	}

	// create pkg/controller dir
	controllerDir := filepath.Join(fullProjectPath, "pkg", "controller")
	if err := os.MkdirAll(controllerDir, defaultDirFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", controllerDir, err)
	}

	// generate pkg/controller/controller.go
	controllerFilePath := filepath.Join(controllerDir, "controller.go")
	controllergen := scaffold.NewControllerCodegen()
	buf = &bytes.Buffer{}
	err = controllergen.Render(buf)
	if err != nil {
		log.Fatalf("failed to render the template for (%v): %v", controllerFilePath, err)
	}
	err = writeFileAndPrint(controllerFilePath, buf.Bytes(), defaultFileMode)
	if err != nil {
		log.Fatalf("failed to create %v: %v", controllerFilePath, err)
	}

	// create pkg/build dir
	buildDir := filepath.Join(fullProjectPath, "pkg", "build")
	if err := os.MkdirAll(buildDir, defaultDirFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", buildDir, err)
	}

	// generate pkg/build/Dockerfile
	dockerfilePath := filepath.Join(buildDir, "Dockerfile")
	dockerfilegen := scaffold.NewDockerfileCodegen(&scaffold.DockerfileInput{ProjectName: projectName})
	buf = &bytes.Buffer{}
	err = dockerfilegen.Render(buf)
	if err != nil {
		log.Fatalf("failed to render the template for (%v): %v", dockerfilePath, err)
	}
	err = writeFileAndPrint(dockerfilePath, buf.Bytes(), defaultFileMode)
	if err != nil {
		log.Fatalf("failed to create %v: %v", dockerfilePath, err)
	}

	// generate pkg/build/build.sh
	buildScriptPath := filepath.Join(buildDir, "build.sh")
	buildScriptGen := scaffold.NewbuildCodegen(
		&scaffold.BuildInput{
			ProjectName: projectName,
			ProjectPath: projectPath,
		})
	buf = &bytes.Buffer{}
	err = buildScriptGen.Render(buf)
	if err != nil {
		log.Fatalf("failed to render the template for (%v): %v", buildScriptPath, err)
	}
	err = writeFileAndPrint(buildScriptPath, buf.Bytes(), defaultExecFileMode)
	if err != nil {
		log.Fatalf("failed to create %v: %v", buildScriptPath, err)
	}

	// create pkg/deploy dir
	deployDir := filepath.Join(fullProjectPath, "deploy")
	if err := os.MkdirAll(deployDir, defaultDirFileMode); err != nil {
		log.Fatalf("failed to create %v: %v", deployDir, err)
	}

	// generate pkg/deploy/role.yaml
	rolePath := filepath.Join(deployDir, "role.yaml")
	roleGen := scaffold.NewRoleCodegen(
		&scaffold.RoleInput{
			ProjectName: projectName,
		})
	buf = &bytes.Buffer{}
	err = roleGen.Render(buf)
	if err != nil {
		log.Fatalf("failed to render the template for (%v): %v", rolePath, err)
	}
	err = writeFileAndPrint(rolePath, buf.Bytes(), defaultFileMode)
	if err != nil {
		log.Fatalf("failed to create %v: %v", rolePath, err)
	}

	// generate pkg/deploy/role_binding.yaml
	roleBindingPath := filepath.Join(deployDir, "role_binding.yaml")
	roleBindingGen := scaffold.NewRoleBindingCodegen(
		&scaffold.RoleBindingInput{
			ProjectName: projectName,
		})
	buf = &bytes.Buffer{}
	err = roleBindingGen.Render(buf)
	if err != nil {
		log.Fatalf("failed to render the template for (%v): %v", roleBindingPath, err)
	}
	err = writeFileAndPrint(roleBindingPath, buf.Bytes(), defaultFileMode)
	if err != nil {
		log.Fatalf("failed to create %v: %v", roleBindingPath, err)
	}

	// TODO: generate rest of the scaffold.
}

// Writes file to a given path and data buffer, as well as prints out a message confirming creation of a file
func writeFileAndPrint(filePath string, data []byte, fileMode os.FileMode) error {
	if err := ioutil.WriteFile(filePath, data, fileMode); err != nil {
		return err
	}
	fmt.Printf("Create %v \n", filePath)
	return nil
}

// repoPath checks if this project's repository path is rooted under $GOPATH and returns project's repository path.
func repoPath() string {
	gp := os.Getenv(gopath)
	if len(gp) == 0 {
		log.Fatal("$GOPATH env not set")
	}
	wd := mustGetwd()
	// check if this project's repository path is rooted under $GOPATH
	if !strings.HasPrefix(wd, gp) {
		log.Fatalf("project's repository path (%v) is not rooted under GOPATH (%v)", wd, gp)
	}
	// compute the repo path by stripping "$GOPATH/src/" from the path of the current directory.
	rp := filepath.Join(string(wd[len(filepath.Join(gp, src)):]), projectName)
	// strip any "/" prefix from the repo path.
	return strings.TrimPrefix(rp, string(filepath.Separator))
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("failed to determine the full path of the current directory: %v", err)
	}
	return wd
}

func execCmd(stdout *os.File, cmd string, args ...string) {
	dc := exec.Command(cmd, args...)
	dc.Dir = filepath.Join(mustGetwd(), projectName)
	dc.Stdout = stdout
	dc.Stderr = os.Stderr
	err := dc.Run()
	if err != nil {
		log.Fatalf("failed to exec %s %#v: %v", cmd, args, err)
	}
}

func pullDep() {
	_, err := exec.LookPath(dep)
	if err != nil {
		log.Fatalf("looking for dep in $PATH: %v", err)
	}
	fmt.Fprintln(os.Stdout, "Run dep ensure ...")
	execCmd(os.Stdout, dep, ensureCmd, "-v")
	fmt.Fprintln(os.Stdout, "Run dep ensure done")
}

func initGit() {
	if skipGit {
		return
	}
	fmt.Fprintln(os.Stdout, "Run git init ...")
	execCmd(os.Stdout, "git", "init")
	execCmd(os.Stdout, "git", "add", "--all")
	execCmd(os.Stdout, "git", "commit", "-q", "-m", "INITIAL COMMIT")
	fmt.Fprintln(os.Stdout, "Run git init done")
}
