package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cmdError "github.com/coreos/operator-sdk/commands/operator-sdk/error"
	"github.com/coreos/operator-sdk/pkg/generator"

	"github.com/spf13/cobra"
)

func NewNewCmd() *cobra.Command {
	newCmd := &cobra.Command{
		Use:   "new <project-name> [required-flags]",
		Short: "Creates a new operator application",
		Long: `The operator-sdk new command creates a new operator application and 
generates a default directory layout based on the input <project-name>. 

<project-name> is the project name of the new operator. (e.g app-operator)

	--api-version and --kind are required flags to generate the new operator application.

For example:
	$ mkdir $GOPATH/src/github.com/example.com/
	$ cd $GOPATH/src/github.com/example.com/
	$ operator-sdk new app-operator --api-group=app.example.com --kind=AppService
generates a skeletal app-operator application in $GOPATH/src/github.com/example.com/app-operator.
`,
		Run: newFunc,
	}

	newCmd.Flags().StringVar(&apiVersion, "api-version", "", "Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)")
	newCmd.MarkFlagRequired("api-version")
	newCmd.Flags().StringVar(&kind, "kind", "", "Kubernetes CustomResourceDefintion kind. (e.g AppService)")
	newCmd.MarkFlagRequired("kind")

	return newCmd
}

var (
	apiVersion  string
	kind        string
	projectName string
)

const (
	gopath    = "GOPATH"
	src       = "src"
	dep       = "dep"
	ensureCmd = "ensure"
)

func newFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, fmt.Errorf("new command needs 1 argument."))
	}
	parse(args)
	verifyFlags()
	g := generator.NewGenerator(apiVersion, kind, projectName, repoPath())
	err := g.Render()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to create project %v: %v", projectName, err))
	}
	pullDep()
}

func parse(args []string) {
	projectName = args[0]
	if len(projectName) == 0 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, fmt.Errorf("project-name must not be empty"))
	}
}

// repoPath checks if this project's repository path is rooted under $GOPATH and returns project's repository path.
func repoPath() string {
	gp := os.Getenv(gopath)
	if len(gp) == 0 {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("$GOPATH env not set"))
	}
	wd := mustGetwd()
	// check if this project's repository path is rooted under $GOPATH
	if !strings.HasPrefix(wd, gp) {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("project's repository path (%v) is not rooted under GOPATH (%v)", wd, gp))
	}
	// compute the repo path by stripping "$GOPATH/src/" from the path of the current directory.
	rp := filepath.Join(string(wd[len(filepath.Join(gp, src)):]), projectName)
	// strip any "/" prefix from the repo path.
	return strings.TrimPrefix(rp, string(filepath.Separator))
}

func verifyFlags() {
	// TODO: verify format of input flags.
}

func pullDep() {
	dc := exec.Command(dep, ensureCmd)
	dc.Dir = filepath.Join(mustGetwd(), projectName)
	o, err := dc.CombinedOutput()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to ensure dependencies: (%v)", string(o)))
	}
	fmt.Fprintln(os.Stdout, string(o))
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to determine the full path of the current directory: %v", err))
	}
	return wd
}
