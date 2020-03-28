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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/flags"

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type bundleValidateCmd struct {
	bundleCmd
}

// newValidateCmd returns a command that will validate an operator bundle image.
func newValidateCmd() *cobra.Command {
	c := bundleValidateCmd{}
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate an operator bundle image",
		Long: `The 'operator-sdk bundle validate' command can validate both content and
format of an operator bundle image or an operator bundles directory on-disk
containing operator metadata and manifests. This command will exit with a non-zero
exit code if any validation tests fail.

Note: if validating an image, the image must exist in a remote registry, not
just locally.
`,
		Example: `The following command flow will generate test-operator bundle image manifests
and validate them, assuming a bundle for 'test-operator' version v0.1.0 exists at
<project-root>/deploy/olm-catalog/test-operator/0.1.0:

  # Generate manifests locally.
  $ operator-sdk bundle create \
      --generate-only \
      --directory ./deploy/olm-catalog/test-operator/0.1.0

  # Validate the directory containing manifests and metadata.
  $ operator-sdk bundle validate ./deploy/olm-catalog/test-operator

To build and validate an image:

  # Build and push the image using the docker CLI.
	$ operator-sdk bundle create quay.io/example/test-operator:v0.1.0 \
      --directory ./deploy/olm-catalog/test-operator/0.1.0
  $ docker push quay.io/example/test-operator:v0.1.0

  # Ensure the image with modified metadata and Dockerfile is valid.
  $ operator-sdk bundle validate quay.io/example/test-operator:v0.1.0

`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if err = c.validate(args); err != nil {
				return fmt.Errorf("error validating args: %v", err)
			}
			// If the argument isn't a directory, assume it's an image.
			if isExist(args[0]) {
				if c.directory, err = relDir(args[0]); err != nil {
					log.Fatal(err)
				}
			} else {
				c.imageTag = args[0]
			}
			if err = c.run(); err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	c.addToFlagSet(cmd.Flags())

	return cmd
}

func (c bundleValidateCmd) validate(args []string) error {
	if len(args) != 1 {
		return errors.New("an image tag or directory is a required argument")
	}
	return nil
}

func (c *bundleValidateCmd) addToFlagSet(fs *pflag.FlagSet) {
	fs.StringVarP(&c.imageBuilder, "image-builder", "b", "docker",
		"Tool to extract container images. One of: [docker, podman]")
}

func (c bundleValidateCmd) run() (err error) {
	// Set directory, either supplied directly or a temp dir used to unpack
	// the image.
	dir := c.directory
	if c.imageTag != "" {
		dir, err = ioutil.TempDir("", "bundle-")
		if err != nil {
			return err
		}
		defer func() {
			if err = os.RemoveAll(dir); err != nil {
				log.Errorf("Error removing temp bundle dir: %v", err)
			}
		}()
	}
	if dir, err = filepath.Abs(dir); err != nil {
		return err
	}

	// Set up logger.
	fields := log.Fields{"bundle-dir": dir}
	if c.imageTag != "" {
		fields["container-tool"] = c.imageBuilder
	}
	logger := log.WithFields(fields)
	if viper.GetBool(flags.VerboseOpt) {
		log.SetLevel(log.DebugLevel)
	}

	val := bundle.NewImageValidator(c.imageBuilder, logger)

	// Pull image if a tag was passed.
	if c.imageTag != "" {
		logger.Info("Unpacked image layers")
		err = val.PullBundleImage(c.imageTag, dir)
		if err != nil {
			logger.Fatalf("Error to unpacking image: %v", err)
		}
	}

	// Validate bundle format.
	if err = val.ValidateBundleFormat(dir); err != nil {
		logger.Fatalf("Bundle format validation failed: %v", err)
	}

	// Validate bundle content.
	manifestsDir := filepath.Join(dir, bundle.ManifestsDir)
	if err = val.ValidateBundleContent(manifestsDir); err != nil {
		logger.Fatalf("Bundle content validation failed: %v", err)
	}

	logger.Info("All validation tests have completed successfully")

	return nil
}
