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

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	apierrors "github.com/operator-framework/api/pkg/validation/errors"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/bundle/internal"
	"github.com/operator-framework/operator-sdk/internal/flags"
	internalregistry "github.com/operator-framework/operator-sdk/internal/registry"
)

type bundleValidateCmd struct {
	bundleCmd

	outputFormat string
}

// newValidateCmd returns a command that will validate an operator bundle image.
func newValidateCmd() *cobra.Command {
	c := bundleValidateCmd{}
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate an operator bundle image",
		Long: `The 'operator-sdk bundle validate' command can validate both content and
format of an operator bundle image or an operator bundles directory on-disk
containing operator metadata and manifests. This command will exit with an
exit code of 1 if any validation errors arise, and 0 if only warnings arise or
all validators pass.

More information about operator bundles and metadata:
https://github.com/operator-framework/operator-registry#manifest-format.

NOTE: if validating an image, the image must exist in a remote registry, not
just locally.
`,
		Example: `The following command flow will generate test-operator bundle image manifests
and validate them, assuming a bundle for 'test-operator' version v0.1.0 exists at
<project-root>/deploy/olm-catalog/test-operator/manifests:

  # Generate manifests locally.
  $ operator-sdk bundle create --generate-only

  # Validate the directory containing manifests and metadata.
  $ operator-sdk bundle validate ./deploy/olm-catalog/test-operator

To build and validate an image:

  # Create a registry namespace or use an existing one.
  $ export NAMESPACE=<your registry namespace>

  # Build and push the image using the docker CLI.
  $ operator-sdk bundle create quay.io/$NAMESPACE/test-operator:v0.1.0
  $ docker push quay.io/$NAMESPACE/test-operator:v0.1.0

  # Ensure the image with modified metadata and Dockerfile is valid.
  $ operator-sdk bundle validate quay.io/$NAMESPACE/test-operator:v0.1.0

`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if viper.GetBool(flags.VerboseOpt) {
				log.SetLevel(log.DebugLevel)
			}

			// Always print non-output logs to stderr as to not pollute actual command output.
			// Note that it allows the JSON result be redirected to the Stdout. E.g
			// if we run the command with `| jq . > result.json` the command will print just the logs
			// and the file will have only the JSON result.
			logger := log.NewEntry(internal.NewLoggerTo(os.Stderr))

			if err = c.validate(args); err != nil {
				return fmt.Errorf("invalid command args: %v", err)
			}

			// If the argument isn't a directory, assume it's an image.
			if isExist(args[0]) {
				if c.directory, err = relWd(args[0]); err != nil {
					logger.Fatal(err)
				}
			} else {
				c.directory, err = ioutil.TempDir("", "bundle-")
				if err != nil {
					return err
				}
				defer func() {
					if err = os.RemoveAll(c.directory); err != nil {
						logger.Errorf("Error removing temp bundle dir: %v", err)
					}
				}()

				logger.Info("Unpacking image layers")

				if err := c.unpackImageIntoDir(args[0], c.directory); err != nil {
					logger.Fatalf("Error unpacking image %s: %v", args[0], err)
				}
			}

			result := c.run()
			if err := result.PrintWithFormat(c.outputFormat); err != nil {
				logger.Fatal(err)
			}

			logger.Info("All validation tests have completed successfully")

			return nil
		},
	}

	c.addToFlagSet(cmd.Flags())

	return cmd
}

// validate verifies the command args
func (c bundleValidateCmd) validate(args []string) error {
	if len(args) != 1 {
		return errors.New("an image tag or directory is a required argument")
	}
	if c.outputFormat != internal.JSONAlpha1 && c.outputFormat != internal.Text {
		return fmt.Errorf("invalid value for output flag: %v", c.outputFormat)
	}
	return nil
}

// TODO: add a "permissive" flag to toggle whether warnings also cause a non-zero
// exit code to be returned (true by default).
func (c *bundleValidateCmd) addToFlagSet(fs *pflag.FlagSet) {
	fs.StringVarP(&c.imageBuilder, "image-builder", "b", "docker",
		"Tool to extract bundle image data. Only used when validating a bundle image. "+
			"One of: [docker, podman]")
	fs.StringVarP(&c.outputFormat, "output", "o", internal.Text,
		"Result format for results. One of: [text, json-alpha1]")

	// It is hidden because it is an alpha option
	// The idea is the next versions of Operator Registry will return a List of errors
	if err := fs.MarkHidden("output"); err != nil {
		panic(err)
	}
}

func (c bundleValidateCmd) run() (res internal.Result) {
	// Create Result to be outputted
	res = internal.NewResult()

	logger := log.WithFields(log.Fields{
		"bundle-dir":     c.directory,
		"container-tool": c.imageBuilder,
	})
	val := registrybundle.NewImageValidator(c.imageBuilder, logger)

	// Validate bundle format.
	if err := val.ValidateBundleFormat(c.directory); err != nil {
		res.AddError(fmt.Errorf("error validating format in %s: %v", c.directory, err))
	}

	// Validate bundle content.
	// TODO(estroz): instead of using hard-coded 'manifests', look up bundle
	// dir name in metadata labels.
	manifestsDir := filepath.Join(c.directory, registrybundle.ManifestsDir)
	results, err := validateBundleContent(logger, manifestsDir)
	if err != nil {
		res.AddError(fmt.Errorf("error validating content in %s: %v", manifestsDir, err))
	}

	// Check the Results will check the []apierrors.ManifestResult returned
	// from the ValidateBundleContent to add the output(s) into the result
	checkResults(results, &res)

	return res
}

// unpackImageIntoDir writes files in image layers found in image imageTag to dir.
func (c bundleValidateCmd) unpackImageIntoDir(imageTag, dir string) error {
	logger := log.WithFields(log.Fields{
		"bundle-dir":     dir,
		"container-tool": c.imageBuilder,
	})
	val := registrybundle.NewImageValidator(c.imageBuilder, logger)

	return val.PullBundleImage(imageTag, dir)
}

// validateBundleContent validates a bundle in manifestsDir.
func validateBundleContent(logger *log.Entry, manifestsDir string) ([]apierrors.ManifestResult, error) {
	// Detect mediaType.
	mediaType, err := registrybundle.GetMediaType(manifestsDir)
	if err != nil {
		return nil, err
	}
	// Read the bundle.
	bundle, err := apimanifests.GetBundleFromDir(manifestsDir)
	if err != nil {
		return nil, err
	}

	return internalregistry.ValidateBundleContent(logger, bundle, mediaType), nil
}

// checkResults logs warnings and errors in results, and returns true if at
// least one error was encountered.
func checkResults(results []apierrors.ManifestResult, res *internal.Result) {
	for _, r := range results {
		for _, w := range r.Warnings {
			res.AddWarn(w)
		}
		for _, e := range r.Errors {
			res.AddError(e)
		}
	}
}
