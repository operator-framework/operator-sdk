package cmd

import (
	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/up"

	"github.com/spf13/cobra"
)

func NewUpCmd() *cobra.Command {
	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Launches the operator",
		Long: `The up command has subcommands that can launch the operator in various ways.
`,
	}

	upCmd.AddCommand(up.NewLocalCmd())
	return upCmd
}
