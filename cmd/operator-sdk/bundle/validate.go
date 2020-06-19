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
	"github.com/operator-framework/operator-registry/pkg/containertools"
	registryimage "github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/operator-framework/operator-registry/pkg/image/execregistry"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/operator-framework/operator-sdk/cmd/operator-sdk/bundle/internal"
	"github.com/operator-framework/operator-sdk/internal/flags"
	internalregistry "github.com/operator-framework/operator-sdk/internal/registry"
)

const (
	longHelp = `The 'operator-sdk bundle validate' command can validate both content and format of an operator bundle
image or an operator bundle directory on-disk containing operator metadata and manifests. This command will exit
with an exit code of 1 if any validation errors arise, and 0 if only warnings arise or all validators pass.

More information about operator bundles and metadata:
https://github.com/operator-framework/operator-registry/blob/master/docs/design/operator-bundle.md

NOTE: if validating an image, the image must exist in a remote registry, not just locally.
`

	examples = `The following command flow will generate test-operator bundle manifests and metadata,
then validate them for 'test-operator' version v0.1.0:

  # Generate manifests and metadata locally.
  $ make bundle

  # Validate the directory containing manifests and metadata.
  $ operator-sdk bundle validate ./bundle

To build and validate an image built with the above manifests and metadata:

  # Create a registry namespace or use an existing one.
  $ export NAMESPACE=<your registry namespace>

  # Build and push the image using the docker CLI.
  $ docker build -f bundle.Dockerfile -t quay.io/$NAMESPACE/test-operator:v0.1.0 .
  $ docker push quay.io/$NAMESPACE/test-operator:v0.1.0

  # Ensure the image with modified metadata and Dockerfile is valid.
  $ operator-sdk bundle validate quay.io/$NAMESPACE/test-operator:v0.1.0
`

	examplesLegacy = `The following command flow will generate test-operator bundle manifests and metadata,
then validate them for 'test-operator' version v0.1.0:

  # Generate manifests and metadata locally.
  $ operator-sdk generate bundle --version 0.1.0

  # Validate the directory containing manifests and metadata.
  $ operator-sdk bundle validate ./deploy/olm-catalog/test-operator

To build and validate an image built with the above manifests and metadata:

  # Create a registry namespace or use an existing one.
  $ export NAMESPACE=<your registry namespace>

  # Build and push the image using the docker CLI.
  $ docker build -f bundle.Dockerfile -t quay.io/$NAMESPACE/test-operator:v0.1.0 .
  $ docker push quay.io/$NAMESPACE/test-operator:v0.1.0

  # Ensure the image with modified metadata and Dockerfile is valid.
  $ operator-sdk bundle validate quay.io/$NAMESPACE/test-operator:v0.1.0
`
)

type bundleValidateCmd struct {
	bundleCmd

	outputFormat string
}

// newValidateCmd returns a command that will validate an operator bundle.
func newValidateCmd() *cobra.Command {
	cmd := makeValidateCmd()
	cmd.Long = longHelp
	cmd.Example = examples
	return cmd
}

// newValidateCmdLegacy returns a command that will validate an operator bundle for the legacy CLI.
func newValidateCmdLegacy() *cobra.Command {
	cmd := makeValidateCmd()
	cmd.Long = longHelp
	cmd.Example = examplesLegacy
	return cmd
}

// makeValidateCmd makes a command that will validate an operator bundle. Help text must be customized.
func makeValidateCmd() *cobra.Command {
	c := bundleValidateCmd{}
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate an operator bundle",
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

			result, err := c.run(logger, args[0])
			if err != nil {
				logger.Fatal(err)
			}
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
		"Tool to pull and unpack bundle images. Only used when validating a bundle image. "+
			"One of: [docker, podman, none]")

	fs.StringVarP(&c.outputFormat, "output", "o", internal.Text,
		"Result format for results. One of: [text, json-alpha1]")
	// It is hidden because it is an alpha option
	// The idea is the next versions of Operator Registry will return a List of errors
	if err := fs.MarkHidden("output"); err != nil {
		panic(err)
	}
}

func (c bundleValidateCmd) run(logger *log.Entry, bundle string) (res internal.Result, err error) {
	// Create a registry to validate bundle files and optionally unpack the image with.
	reg, err := newImageRegistryForTool(logger, c.imageBuilder)
	if err != nil {
		return res, fmt.Errorf("error creating image registry: %v", err)
	}
	defer func() {
		if err := reg.Destroy(); err != nil {
			logger.Errorf("Error destroying image registry: %v", err)
		}
	}()

	// If bundle isn't a directory, assume it's an image.
	if isExist(bundle) {
		if c.directory, err = relWd(bundle); err != nil {
			return res, err
		}
	} else {
		c.directory, err = ioutil.TempDir("", "bundle-")
		if err != nil {
			return res, err
		}
		defer func() {
			if err = os.RemoveAll(c.directory); err != nil {
				logger.Errorf("Error removing temp bundle dir: %v", err)
			}
		}()

		logger.Info("Unpacking image layers")

		if err := c.unpackImageIntoDir(reg, bundle, c.directory); err != nil {
			return res, fmt.Errorf("error unpacking image %s: %v", bundle, err)
		}
	}

	// Create Result to be outputted
	res = internal.NewResult()

	logger = logger.WithFields(log.Fields{
		"bundle-dir":     c.directory,
		"container-tool": c.imageBuilder,
	})
	val := registrybundle.NewImageValidator(reg, logger)

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

	return res, nil
}

// newImageRegistryForTool returns an image registry based on what type of image tool is passed.
// If toolStr is empty, a containerd registry is returned.
func newImageRegistryForTool(logger *log.Entry, toolStr string) (reg registryimage.Registry, err error) {
	switch toolStr {
	case containertools.DockerTool.String():
		reg, err = execregistry.NewRegistry(containertools.DockerTool, logger)
	case containertools.PodmanTool.String():
		reg, err = execregistry.NewRegistry(containertools.PodmanTool, logger)
	case containertools.NoneTool.String():
		reg, err = containerdregistry.NewRegistry(
			containerdregistry.WithLog(logger),
			// In case reg.Destroy() fails in the caller, make it obvious where this cache came from.
			containerdregistry.WithCacheDir(filepath.Join(os.TempDir(), "bundle-validate-cache")),
		)
	default:
		err = fmt.Errorf("unrecognized image-builder option: %s", toolStr)
	}
	return reg, err
}

// unpackImageIntoDir writes files in image layers found in image imageTag to dir.
func (c bundleValidateCmd) unpackImageIntoDir(reg registryimage.Registry, imageTag, dir string) error {
	logger := log.WithFields(log.Fields{
		"bundle-dir":     dir,
		"container-tool": c.imageBuilder,
	})
	val := registrybundle.NewImageValidator(reg, logger)

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
