package bundle

import (
	"github.com/spf13/cobra"
)

type bundleCmd struct {
	directory      string
	imageTag       string
	imageBuilder   string
	packageName    string
	channels       string
	channelDefault string
	overwrite      bool
}

func NewCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "bundle",
		Short: "Operator bundle commands",
		Long:  `Generate operator bundle metadata and build bundle image.`,
	}

	runCmd.AddCommand(
		newBundleBuildCmd(),
		newBundleGenerateCmd(),
	)
	return runCmd
}
