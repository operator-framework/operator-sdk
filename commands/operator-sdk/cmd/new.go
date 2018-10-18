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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/cmdutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/ansible"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
	rbacv1 "k8s.io/api/rbac/v1"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
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

	newCmd.Flags().StringVar(&apiVersion, "api-version", "", "Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)")
	newCmd.Flags().StringVar(&kind, "kind", "", "Kubernetes CustomResourceDefintion kind. (e.g AppService)")
	newCmd.Flags().StringVar(&operatorType, "type", "go", "Type of operator to initialize (e.g \"ansible\")")
	newCmd.Flags().BoolVar(&skipGit, "skip-git-init", false, "Do not init the directory as a git repository")
	newCmd.Flags().BoolVar(&generatePlaybook, "generate-playbook", false, "Generate a playbook skeleton. (Only used for --type ansible)")

	return newCmd
}

var (
	apiVersion       string
	kind             string
	operatorType     string
	projectName      string
	skipGit          bool
	generatePlaybook bool
)

const (
	gopath              = "GOPATH"
	src                 = "src"
	dep                 = "dep"
	ensureCmd           = "ensure"
	goOperatorType      = "go"
	ansibleOperatorType = "ansible"
)

func newFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		log.Fatal("new command needs 1 argument")
	}
	parse(args)
	mustBeNewProject()
	verifyFlags()
	switch operatorType {
	case goOperatorType:
		doScaffold()
		pullDep()
	case ansibleOperatorType:
		doAnsibleScaffold()
	}
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
		Repo:           filepath.Join(cmdutil.CheckAndGetCurrPkg(), projectName),
		AbsProjectPath: filepath.Join(cmdutil.MustGetwd(), projectName),
		ProjectName:    projectName,
	}

	s := &scaffold.Scaffold{}
	err := s.Execute(cfg,
		&scaffold.Cmd{},
		&scaffold.Dockerfile{},
		&scaffold.ServiceAccount{},
		&scaffold.Role{},
		&scaffold.RoleBinding{},
		&scaffold.Operator{},
		&scaffold.Apis{},
		&scaffold.Controller{},
		&scaffold.Version{},
		&scaffold.Gitignore{},
		&scaffold.GopkgToml{},
	)
	if err != nil {
		log.Fatalf("new scaffold failed: (%v)", err)
	}
}

func doAnsibleScaffold() {
	cfg := &input.Config{
		AbsProjectPath: filepath.Join(cmdutil.MustGetwd(), projectName),
		ProjectName:    projectName,
	}

	resource, err := scaffold.NewResource(apiVersion, kind)
	if err != nil {
		log.Fatal("Invalid apiVersion and kind.")
	}

	s := &scaffold.Scaffold{}
	tmpdir, err := ioutil.TempDir("", "osdk")
	if err != nil {
		log.Fatal("unable to get temp directory")
	}

	galaxyInit := &ansible.GalaxyInit{
		Resource: *resource,
		Dir:      tmpdir,
	}

	err = s.Execute(cfg,
		&ansible.Dockerfile{
			GeneratePlaybook: generatePlaybook,
		},
		&ansible.WatchesYAML{
			Resource:         *resource,
			GeneratePlaybook: generatePlaybook,
		},
		galaxyInit,
		&scaffold.ServiceAccount{},
		&scaffold.Role{},
		&scaffold.RoleBinding{},
		&ansible.Operator{},
		&scaffold.Crd{
			Resource: resource,
		},
		&scaffold.Cr{
			Resource: resource,
		},
	)
	if err != nil {
		log.Fatalf("new scaffold failed: (%v)", err)
	}

	// Decide on playbook.
	if generatePlaybook {
		err := s.Execute(cfg,
			&ansible.Playbook{
				Resource: *resource,
			},
		)
		if err != nil {
			log.Fatalf("new scaffold failed: (%v)", err)
		}
	}

	// Run galaxy init.
	cmd := exec.Command(filepath.Join(galaxyInit.AbsProjectPath, galaxyInit.Path))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
	// Delete Galxy INIT
	// Mac OS tmp directory is /var/folders/_c/..... this means we have to make sure that we get the top level directory to remove
	// everything.
	tmpDirectorySlice := strings.Split(os.TempDir(), "/")
	if err = os.RemoveAll(filepath.Join(galaxyInit.AbsProjectPath, tmpDirectorySlice[1])); err != nil {
		log.Fatalf("failed to remove the galaxy init script")
	}

	// update deploy/role.yaml for the given resource r.
	if err := updateRoleForResource(resource, cfg.AbsProjectPath); err != nil {
		log.Fatalf("failed to update the RBAC manifest for the resource (%v, %v): %v", resource.APIVersion, resource.Kind, err)
	}
}

// repoPath checks if this project's repository path is rooted under $GOPATH and returns project's repository path.
// repoPath field on generator is used primarily in generation of Go operator. For Ansible we will set it to cwd
func repoPath() string {
	// We only care about GOPATH constraint checks if we are a Go operator
	wd := cmdutil.MustGetwd()
	if operatorType == goOperatorType {
		gp := os.Getenv(gopath)
		if len(gp) == 0 {
			log.Fatal("$GOPATH env not set")
		}
		// check if this project's repository path is rooted under $GOPATH
		if !strings.HasPrefix(wd, gp) {
			log.Fatalf("project's repository path (%v) is not rooted under GOPATH (%v)", wd, gp)
		}
		// compute the repo path by stripping "$GOPATH/src/" from the path of the current directory.
		rp := filepath.Join(string(wd[len(filepath.Join(gp, src)):]), projectName)
		// strip any "/" prefix from the repo path.
		return strings.TrimPrefix(rp, string(filepath.Separator))
	}
	return wd
}

func verifyFlags() {
	if operatorType != goOperatorType && operatorType != ansibleOperatorType {
		log.Fatal("--type can only be `go` or `ansible`")
	}
	if operatorType != ansibleOperatorType && generatePlaybook {
		log.Fatal("--generate-playbook can only be used with --type `ansible`")
	}
	if operatorType == goOperatorType && (len(apiVersion) != 0 || len(kind) != 0) {
		log.Fatal(`go type operator does not use --api-version or --kind. Please see "operator-sdk add" command after running new.`)
	}

	if operatorType != goOperatorType {
		if len(apiVersion) == 0 {
			log.Fatal("--api-version must not have empty value")
		}
		if len(kind) == 0 {
			log.Fatal("--kind must not have empty value")
		}
		kindFirstLetter := string(kind[0])
		if kindFirstLetter != strings.ToUpper(kindFirstLetter) {
			log.Fatal("--kind must start with an uppercase letter")
		}
		if strings.Count(apiVersion, "/") != 1 {
			log.Fatalf("api-version has wrong format (%v); format must be $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)", apiVersion)
		}
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
	execCmd(os.Stdout, "git", "commit", "-q", "-m", "INITIAL COMMIT")
	fmt.Fprintln(os.Stdout, "Run git init done")
}

// Copied from add/api.go command
func updateRoleForResource(r *scaffold.Resource, absProjectPath string) error {
	// append rbac rule to deploy/role.yaml
	roleFilePath := filepath.Join(absProjectPath, "deploy", "role.yaml")
	roleYAML, err := ioutil.ReadFile(roleFilePath)
	if err != nil {
		return fmt.Errorf("failed to read role manifest %v: %v", roleFilePath, err)
	}
	obj, _, err := cgoscheme.Codecs.UniversalDeserializer().Decode(roleYAML, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to decode role manifest %v: %v", roleFilePath, err)
	}
	switch role := obj.(type) {
	// TODO: handle cluster roles for operators to watch every namespace.
	case *rbacv1.Role:
		pr := &rbacv1.PolicyRule{}
		apiGroupFound := false
		for i := range role.Rules {
			if role.Rules[i].APIGroups[0] == r.FullGroup {
				apiGroupFound = true
				pr = &role.Rules[i]
				break
			}
		}
		// check if the resource already exists
		for _, resource := range pr.Resources {
			if resource == r.Resource {
				log.Printf("deploy/role.yaml RBAC rules already up to date for the resource (%v, %v)", r.APIVersion, r.Kind)
				return nil
			}
		}

		pr.Resources = append(pr.Resources, r.Resource)
		// create a new apiGroup if not found.
		if !apiGroupFound {
			pr.APIGroups = []string{r.FullGroup}
			// Using "*" to allow access to the resource and all its subresources e.g "memcacheds" and "memcacheds/finalizers"
			// https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#ownerreferencespermissionenforcement
			pr.Resources = []string{"*"}
			pr.Verbs = []string{"*"}
			role.Rules = append(role.Rules, *pr)
		}
		// update role.yaml
		d, err := json.Marshal(&role)
		if err != nil {
			return fmt.Errorf("failed to marshal role(%+v): %v", role, err)
		}
		m := &map[string]interface{}{}
		err = yaml.Unmarshal(d, m)
		data, err := yaml.Marshal(m)
		if err != nil {
			return fmt.Errorf("failed to marshal role(%+v): %v", role, err)
		}
		if err := ioutil.WriteFile(roleFilePath, data, cmdutil.DefaultFileMode); err != nil {
			return fmt.Errorf("failed to update %v: %v", roleFilePath, err)
		}
	default:
		return errors.New("failed to parse role.yaml as a role")
	}
	// not reachable
	return nil
}
