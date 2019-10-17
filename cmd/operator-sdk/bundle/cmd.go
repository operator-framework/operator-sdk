package main

import (
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "bundle",
		Short: "Operator bundle commands",
		Long:  `Generate operator bundle metadata and build bundle image.`,
	}

	runCmd.AddCommand(newBundleGenerateCmd())
	runCmd.AddCommand(newBundleBuildCmd())
	return runCmd
}
