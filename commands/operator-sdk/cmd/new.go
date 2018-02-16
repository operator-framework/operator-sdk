package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newCmd represents the new command
var newCmd = &cobra.Command{
	Use:   "new [options] <project-name>",
	Short: "Creates a new operator application",
	Long: `The operator-sdk new command creates a new operator application and 
	generates a default directory layout based on the input <project-name>. 

	If none of options are specified, the operator-sdk defaults 
	Kubernetes api group to <project-name>.example.com/v1 
	and Kubernetes resource kind to <project-name>Service.

	For example,
	$ mkdir $GOPATH/src/github.com/example.com/play
	$ cd $GOPATH/src/github.com/example.com/play
	$ operator-sdk new play
	generates a skeletal play application in $GOPATH/src/github.com/example.com/play.
`,
	Run: newFunc,
}

var (
	apiGroup string
	kind     string
)

func init() {
	RootCmd.AddCommand(newCmd)
	newCmd.Flags().StringVar(&apiGroup, "api-group", "play.example.com/v1", "Kubernetes API Group. e.g play.example.com/v1")
	newCmd.Flags().StringVar(&kind, "kind", "PlayService", "Kubernetes Resource Kind. e.g PlayService")
}

func newFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		ExitWithError(ExitBadArgs, fmt.Errorf("new command needs 1 arguments."))
	}
	parse(args)
	// TODO: add generation logic.
}

func parse(args []string) {
}
