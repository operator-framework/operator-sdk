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
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/cmdutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

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
	dep       = "dep"
	ensureCmd = "ensure"
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
	fp := filepath.Join(cmdutil.MustGetwd(), projectName)
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

	cfg := &input.Config{
		Repo:        filepath.Join(cmdutil.CheckAndGetCurrPkg(), projectName),
		ProjectPath: filepath.Join(cmdutil.MustGetwd(), projectName),
		ProjectName: projectName,
	}

	s := &scaffold.Scaffold{}
	err := s.Execute(cfg,
		&scaffold.Cmd{},
		&scaffold.Dockerfile{},
		&scaffold.Build{},
		&scaffold.Role{},
		&scaffold.RoleBinding{},
		&scaffold.Operator{},
		&scaffold.Apis{},
		&scaffold.Controller{},
		&scaffold.Version{},
		&scaffold.Gitignore{},
		&scaffold.GopkgToml{},
		&scaffold.Boilerplate{},
	)
	if err != nil {
		log.Fatalf("new scaffold failed: (%v)", err)
	}
}

func execCmd(stdout *os.File, cmd string, args ...string) {
	dc := exec.Command(cmd, args...)
	dc.Dir = filepath.Join(cmdutil.MustGetwd(), projectName)
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
	execCmd(nil, "git", "commit", "-m", "INITIAL COMMIT")
	fmt.Fprintln(os.Stdout, "Run git init done")
}
