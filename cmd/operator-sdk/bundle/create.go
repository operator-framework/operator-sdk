// Copyright 2020 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bundle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	catalog "github.com/operator-framework/operator-sdk/internal/scaffold/olm-catalog"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// newCreateCmd returns a command that will build operator bundle image or
// generate metadata for them.
func newCreateCmd() *cobra.Command {
	c := &bundleCmd{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an operator bundle image",
		Long: `The 'operator-sdk bundle create' command will build an operator
bundle image containing operator metadata and manifests, tagged with the
provided image tag.

To write metadata and a bundle image Dockerfile to disk, set '--generate-only=true'.
Bundle metadata will be generated in <directory-arg>/metadata, and the Dockerfile
in <directory-arg>. This flag is useful if you want to build an operator's
bundle image manually or modify metadata before building an image.

More information on operator bundle images and metadata:
https://github.com/openshift/enhancements/blob/master/enhancements/olm/operator-bundle.md#docker

NOTE: bundle images are not runnable.`,
		Example: `The following invocation will build a test-operator bundle image using Docker.
This image will contain manifests for package channels 'stable' and 'beta':

$ operator-sdk bundle create quay.io/example/test-operator:v0.1.0 \
    --directory ./deploy/olm-catalog/test-operator \
    --package test-operator \
    --channels stable,beta \
    --default-channel stable

Assuming your operator has the same name as your operator and the only channel
is 'stable', the above command can be abbreviated to:

$ operator-sdk bundle create quay.io/example/test-operator:v0.1.0

The following invocation will generate test-operator bundle metadata and
Dockerfile without building the image:

$ operator-sdk bundle create \
    --generate-only \
    --directory ./deploy/olm-catalog/test-operator \
    --package test-operator \
    --channels stable,beta \
    --default-channel stable`,
		RunE: func(cmd *cobra.Command, args []string) error {
			channels := strings.Join(c.channels, ",")
			if c.generateOnly {
				if len(args) != 0 {
					return fmt.Errorf("command %s does not accept any arguments", cmd.CommandPath())
				}
				err := bundle.GenerateFunc(c.directory, c.packageName, channels,
					c.defaultChannel, true)
				if err != nil {
					log.Fatalf("Error generating bundle image files: %v", err)
				}
				return nil
			}
			// An image tag is required for build only.
			if len(args) != 1 {
				return errors.New("a bundle image tag is a required argument, ex. example.com/test-operator:v0.1.0")
			}
			c.imageTag = args[0]
			// Clean up transient metadata and Dockerfile once the image is built,
			// as they are no longer needed.
			for _, cleanup := range c.cleanupFuncs() {
				defer cleanup()
			}
			// Build but never overwrite existing metadata/Dockerfile.
			err := bundle.BuildFunc(c.directory, c.imageTag, c.imageBuilder,
				c.packageName, channels, c.defaultChannel, false)
			if err != nil {
				log.Fatalf("Error building bundle image: %v", err)
			}
			return nil
		},
	}

	// Set up default values.
	projectName := filepath.Base(projutil.MustGetwd())
	defaultDir := ""
	if _, err := os.Stat(catalog.OLMCatalogDir); err == nil || os.IsExist(err) {
		defaultDir = filepath.Join(catalog.OLMCatalogDir, projectName)
	}
	defaultChannels := []string{"stable"}

	cmd.Flags().StringVarP(&c.directory, "directory", "d", defaultDir,
		"The directory where bundle manifests are located")
	cmd.Flags().StringVarP(&c.packageName, "package", "p", projectName,
		"The name of the package that bundle image belongs to. Set if package name differs from project name")
	cmd.Flags().StringSliceVarP(&c.channels, "channels", "c", defaultChannels,
		"The list of channels that bundle image belongs to")
	cmd.Flags().BoolVarP(&c.generateOnly, "generate-only", "g", false,
		"Generate metadata and a Dockerfile on disk without building the bundle image")
	cmd.Flags().StringVarP(&c.imageBuilder, "image-builder", "b", "docker",
		"Tool to build container images. One of: [docker, podman, buildah]")
	cmd.Flags().StringVarP(&c.defaultChannel, "default-channel", "e", "",
		"The default channel for the bundle image")

	return cmd
}
