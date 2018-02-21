package cmd

import (
	"fmt"

	"github.com/coreos/operator-sdk/pkg/generator"

	"github.com/spf13/cobra"
)

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new <project-name> [required-flags]",
	Short: "Creates a new operator application",
	Long: `The operator-sdk new command creates a new operator application and 
	generates a default directory layout based on the input <project-name>. 

	<project-name> is the project name of the new operator. (e.g app-operator)

	--api-version and --kind are required flags to generate the new operator application.

	For example,
	$ mkdir $GOPATH/src/github.com/example.com/
	$ cd $GOPATH/src/github.com/example.com/
	$ operator-sdk new app-operator --api-group=app.example.com --kind=AppService
	generates a skeletal app-operator application in $GOPATH/src/github.com/example.com/app-operator.
`,
	Run: newFunc,
}

var (
	apiVersion  string
	kind        string
	projectName string
)

func init() {
	RootCmd.AddCommand(newCmd)
	newCmd.Flags().StringVar(&apiVersion, "api-version", "", "Kubernetes apiVersion and has a format of $GROUP_NAME/$VERSION (e.g app.example.com/v1alpha1)")
	newCmd.MarkFlagRequired("api-version")
	newCmd.Flags().StringVar(&kind, "kind", "", "Kubernetes CustomResourceDefintion kind. (e.g AppService)")
	newCmd.MarkFlagRequired("kind")
}

func newFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		ExitWithError(ExitBadArgs, fmt.Errorf("new command needs 1 argument."))
	}
	parse(args)
	verifyFlags()
	g := generator.NewGenerator(apiVersion, kind, projectName)
	err := g.Render()
	if err != nil {
		ExitWithError(ExitError, fmt.Errorf("failed to create project %v: %v", projectName, err))
	}
}

func parse(args []string) {
	projectName = args[0]
	if len(projectName) == 0 {
		ExitWithError(ExitBadArgs, fmt.Errorf("project-name must not be empty"))
	}
}

func verifyFlags() {
	// TODO: verify format of input flags.
}
