package bundle

import (
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// newBundleGenerateCmd returns a command that will generate operator bundle
// annotations.yaml metadata
func newBundleGenerateCmd() *cobra.Command {
	c := bundleCmd{}
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate operator bundle metadata and Dockerfile",
		Long: `The 'operator-sdk bundle generate' command will generate operator
bundle metadata in <directory-arg>/metadata and a Dockerfile to build an
operator bundle image in <directory-arg>.

Unlike 'build', 'generate' does not build the image, permitting use of a
non-default image builder

NOTE: modifying generated metadata is not recommended and may corrupt the
resulting image.
`,
		Example: `The following command will generate metadata and a Dockerfile defining
a test-operator bundle image containing manifests for package channels
'stable' and 'beta':

$ operator-sdk bundle generate \
    --directory ./deploy/olm-catalog/test-operator \
    --package test-operator \
    --channels stable,beta \
    --default stable
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return bundle.GenerateFunc(c.directory, c.packageName, c.channels,
				c.channelDefault, true)
		},
	}

	cmd.Flags().StringVarP(&c.directory, "directory", "d", "",
		"The directory where bundle manifests are located.")
	if err := cmd.MarkFlagRequired("directory"); err != nil {
		log.Fatalf("Failed to mark `directory` flag for `generate` subcommand as required")
	}
	cmd.Flags().StringVarP(&c.packageName, "package", "p", "",
		"The name of the package that bundle image belongs to")
	if err := cmd.MarkFlagRequired("package"); err != nil {
		log.Fatalf("Failed to mark `package` flag for `generate` subcommand as required")
	}
	cmd.Flags().StringVarP(&c.channels, "channels", "c", "",
		"The list of channels that bundle image belongs to")
	if err := cmd.MarkFlagRequired("channels"); err != nil {
		log.Fatalf("Failed to mark `channels` flag for `generate` subcommand as required")
	}
	cmd.Flags().StringVarP(&c.channelDefault, "default", "e", "",
		"The default channel for the bundle image")

	return cmd
}
