package main

import (
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	defaultPermission = 0644
	registryV1Type    = "registry+v1"
	plainType         = "plain"
	helmType          = "helm"
	manifestsMetadata = "manifests+metadata"
	annotationsFile   = "annotations.yaml"
	dockerFile        = "Dockerfile"
	resourcesLabel    = "operators.operatorframework.io.bundle.resources"
	mediatypeLabel    = "operators.operatorframework.io.bundle.mediatype"
)

type AnnotationMetadata struct {
	Annotations AnnotationType `yaml:"annotations"`
}

type AnnotationType struct {
	Resources string `yaml:"operators.operatorframework.io.bundle.resources"`
	MediaType string `yaml:"operators.operatorframework.io.bundle.mediatype"`
}

// newBundleGenerateCmd returns a command that will generate operator bundle
// annotations.yaml metadata
func newBundleGenerateCmd() *cobra.Command {
	bundleGenerateCmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate operator bundle metadata and Dockerfile",
		Long: `The opm buindle generate command will generate operator
        bundle metadata if needed and a Dockerfile to build Operator bundle image.

        $ operator-sdk bundle generate -d /test/0.1.0/
        `,
		RunE: generateFunc,
	}

	bundleGenerateCmd.Flags().StringVarP(&dirBuildArgs, "directory", "d", "", "The directory where bundle manifests are located.")
	if err := bundleGenerateCmd.MarkFlagRequired("directory"); err != nil {
		log.Fatalf("Failed to mark `directory` flag for `generate` subcommand as required")
	}

	return bundleGenerateCmd
}

func generateFunc(cmd *cobra.Command, args []string) error {
	err := bundle.GenerateFunc(dirBuildArgs)
	if err != nil {
		return err
	}

	return nil
}
