package cmd

import "github.com/spf13/cobra"

// This represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "operator-sdk",
	Short: "A sdk for building operator with ease",
}
