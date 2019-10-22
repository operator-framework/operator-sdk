package bundle

import (
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	dirBuildArgs       string
	tagBuildArgs       string
	imageBuilderArgs   string
	packageNameArgs    string
	channelsArgs       string
	channelDefaultArgs string
	overwriteArgs      bool
)

// newBundleBuildCmd returns a command that will build operator bundle image.
func newBundleBuildCmd() *cobra.Command {
	bundleBuildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build operator bundle image",
		Long: `The "operator-sdk bundle build" command will generate operator
        bundle metadata if needed and build bundle image with operator manifest
        and metadata.

        For example: The command will generate annotations.yaml metadata plus
        Dockerfile for bundle image and then build a container image from
        provided operator bundle manifests generated metadata
        e.g. "quay.io/example/operator:v0.0.1".

        After the build process is completed, a container image would be built
        locally in docker and available to push to a container registry.

        $ operator-sdk bundle build --directory /test/ --tag quay.io/example/operator:v0.1.0 \
		--package test-operator --channels stable,beta --default stable --overwrite

        Note: Bundle image is not runnable.
        `,
		RunE: buildFunc,
	}

	bundleBuildCmd.Flags().StringVarP(&dirBuildArgs, "directory", "d", "", "The directory where bundle manifests are located")
	if err := bundleBuildCmd.MarkFlagRequired("directory"); err != nil {
		log.Fatalf("Failed to mark `directory` flag for `build` subcommand as required")
	}

	bundleBuildCmd.Flags().StringVarP(&tagBuildArgs, "tag", "t", "", "The image tag applied to the bundle image")
	if err := bundleBuildCmd.MarkFlagRequired("tag"); err != nil {
		log.Fatalf("Failed to mark `tag` flag for `build` subcommand as required")
	}

	bundleBuildCmd.Flags().StringVarP(&packageNameArgs, "package", "p", "", "The name of the package that bundle image belongs to")
	if err := bundleBuildCmd.MarkFlagRequired("package"); err != nil {
		log.Fatalf("Failed to mark `package` flag for `build` subcommand as required")
	}

	bundleBuildCmd.Flags().StringVarP(&channelsArgs, "channels", "c", "", "The list of channels that bundle image belongs to")
	if err := bundleBuildCmd.MarkFlagRequired("channels"); err != nil {
		log.Fatalf("Failed to mark `channels` flag for `build` subcommand as required")
	}

	bundleBuildCmd.Flags().StringVarP(&imageBuilderArgs, "image-builder", "b", "docker", "Tool to build container images. One of: [docker, podman, buildah]")

	bundleBuildCmd.Flags().StringVarP(&channelDefaultArgs, "default", "e", "", "The default channel for the bundle image")

	bundleBuildCmd.Flags().BoolVarP(&overwriteArgs, "overwrite", "o", false, "To overwrite annotations.yaml locally if existed. By default, overwrite is set to `false`.")

	return bundleBuildCmd
}

func buildFunc(cmd *cobra.Command, args []string) error {
	err := bundle.BuildFunc(dirBuildArgs, tagBuildArgs, imageBuilderArgs,
		packageNameArgs, channelsArgs, channelDefaultArgs, overwriteArgs)
	if err != nil {
		return err
	}

	return nil
}
