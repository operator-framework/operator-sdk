package main

import (
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	dirBuildArgs     string
	tagBuildArgs     string
	imageBuilderArgs string
)

// newBundleBuildCmd returns a command that will build operator bundle image.
func newBundleBuildCmd() *cobra.Command {
	bundleBuildCmd := &cobra.Command{
		Use:   "build",
		Short: "Build operator bundle image",
		Long: `The operator-sdk bundle build command will generate operator
        bundle metadata if needed and build bundle image with operator manifest
        and metadata.

        For example: The command will generate annotations.yaml metadata plus
        Dockerfile for bundle image and then build a container image from
        provided operator bundle manifests generated metadata
        e.g. "quay.io/example/operator:v0.1.0".

        After the build process is completed, a container image would be built
        locally in docker and available to push to a container registry.

        $ operator-sdk bundle build -dir /test/0.1.0/ -t quay.io/example/operator:v0.1.0

        Note: Bundle image is not runnable.
        `,
		RunE: buildFunc,
	}

	bundleBuildCmd.Flags().StringVarP(&dirBuildArgs, "directory", "d", "", "The directory where bundle manifests are located.")
	if err := bundleBuildCmd.MarkFlagRequired("directory"); err != nil {
		log.Fatalf("Failed to mark `directory` flag for `build` subcommand as required")
	}

	bundleBuildCmd.Flags().StringVarP(&tagBuildArgs, "tag", "t", "", "The name of the bundle image will be built.")
	if err := bundleBuildCmd.MarkFlagRequired("tag"); err != nil {
		log.Fatalf("Failed to mark `tag` flag for `build` subcommand as required")
	}

	bundleBuildCmd.Flags().StringVarP(&imageBuilderArgs, "image-builder", "b", "docker", "Tool to build container images. One of: [docker, podman, buildah]")

	return bundleBuildCmd
}

func buildFunc(cmd *cobra.Command, args []string) error {
	err := bundle.BuildFunc(dirBuildArgs, tagBuildArgs, imageBuilderArgs)
	if err != nil {
		return err
	}

	return nil
}
