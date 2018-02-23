package cmd

import (
	"github.com/coreos/operator-sdk/commands/operator-sdk/cmd/generate"

	"github.com/spf13/cobra"
)

func NewGenerateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate <generator>",
		Short: "Invokes specific generator",
		Long: `The operator-sdk generate command invokes specific generator to generate code as needed.
`,
	}
	cmd.AddCommand(generate.NewGenerateK8SCmd())
	return cmd
}
