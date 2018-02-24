package cmd

import "github.com/spf13/cobra"

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator-sdk",
		Short: "A sdk for building operator with ease",
	}

	cmd.AddCommand(NewNewCmd())
	cmd.AddCommand(NewBuildCmd())
	cmd.AddCommand(NewGenerateCmd())

	return cmd
}
