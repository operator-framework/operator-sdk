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

package validate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/operator-registry/pkg/containertools"
	registryimage "github.com/operator-framework/operator-registry/pkg/image"
	"github.com/operator-framework/operator-registry/pkg/image/containerdregistry"
	"github.com/operator-framework/operator-registry/pkg/image/execregistry"
	registrybundle "github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/labels"

	internalregistry "github.com/operator-framework/operator-sdk/internal/registry"
	"github.com/operator-framework/operator-sdk/internal/validate"
)

type bundleValidateCmd struct {
	directory           string
	imageBuilder        string
	outputFormat        string
	selectorRaw         string
	selector            labels.Selector
	listOptional        bool
	optionalValues      map[string]string
	alphaSelectExternal string
}

// validate verifies the command args
func (c bundleValidateCmd) validate(args []string) error {
	if c.listOptional {
		return nil
	}

	if len(args) != 1 {
		return errors.New("an image tag or directory is a required argument")
	}
	if c.outputFormat != validate.JSONAlpha1Output && c.outputFormat != validate.TextOutput {
		return fmt.Errorf("invalid value for output flag: %v", c.outputFormat)
	}

	// Check optional selector.
	if c.selectorRaw != "" {
		if err := optionalValidators.checkMatches(c.selector); err != nil {
			return err
		}
	}

	return nil
}

// TODO: add a "permissive" flag to toggle whether warnings also cause a non-zero
// exit code to be returned (true by default).
func (c *bundleValidateCmd) addToFlagSet(fs *pflag.FlagSet) {
	fs.StringVarP(&c.imageBuilder, "image-builder", "b", "docker",
		"Tool to pull and unpack bundle images. Only used when validating a bundle image. "+
			"One of: [docker, podman, none]")
	fs.StringVar(&c.selectorRaw, "select-optional", "",
		"Label selector to select optional validators to run. "+
			"Run this command with '--list-optional' to list available optional validators")
	fs.BoolVar(&c.listOptional, "list-optional", false,
		"List all optional validators available. When set, no validators will be run")

	optionalValueEmpty := map[string]string{}
	fs.StringToStringVarP(&c.optionalValues, "optional-values", "", optionalValueEmpty,
		"Inform a []string map of key=values which can be used by the validator. e.g. to check the operator bundle "+
			"against an Kubernetes version that it is intended to be distributed use `--optional-values=k8s-version=1.22`")

	fs.StringVarP(&c.outputFormat, "output", "o", validate.TextOutput,
		"Result format for results. One of: [text, json-alpha1]. Note: output format types containing "+
			"\"alphaX\" are subject to change and not covered by guarantees of stable APIs.")

	// alpha-select-external should be set to a Unix path list
	// ("/path/to/e1.sh:/path/to/e2") containing the list of entrypoints to
	// external (out of code tree) validator scripts or binaries to run.
	// Requirements for entrypoints:
	//  - Entrypoints must be executable by the user running the parent process.
	//  - The stdout output of an entrypoint *must* conform to the JSON
	//    representation of Result so results can be parsed and collated
	//    with other internal validators.
	//  - An entrypoint should exit 1 and print output to stderr only if the
	//    entrypoint itself fails for some reason. If the bundle fails to
	//    pass validation, that information  should be encoded in the Result
	//    printed to stdout as a Type=\"error\".
	//
	// WARNING: the script or binary at the base of this path will be
	// executed arbitrarily, so make sure you check the contents of that
	// script or binary prior to running.
	fs.StringVar(&c.alphaSelectExternal, "alpha-select-external", "",
		"Selector to select external validators to run. It should be set to a Unix path list (\"/path/to/e1.sh:/path/to/e2\")")
}

func (c bundleValidateCmd) run(logger *log.Entry, bundleRaw string) (res *validate.Result, err error) {
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
	if isExist(bundleRaw) {
		if c.directory, err = relWd(bundleRaw); err != nil {
			return res, err
		}
	} else {
		c.directory, err = os.MkdirTemp("", "bundle-")
		if err != nil {
			return res, err
		}
		defer func() {
			if err := os.RemoveAll(c.directory); err != nil {
				logger.Errorf("Error removing temp bundle dir: %v", err)
			}
		}()

		logger.Info("Unpacking image layers")

		if err := c.unpackImageIntoDir(reg, bundleRaw, c.directory); err != nil {
			return res, fmt.Errorf("error unpacking image %s: %v", bundleRaw, err)
		}
	}

	// Read the bundle object and metadata from the created/passed in directory.
	bundle, mediaType, err := getBundleDataFromDir(c.directory)
	if err != nil {
		return res, err
	}

	// Create Result to be output.
	res = validate.NewResult()

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
	results := internalregistry.ValidateBundleContent(logger, bundle, mediaType)
	res.AddManifestResults(results...)

	// Run optional validators.
	results = runOptionalValidators(bundle, c.selector, c.optionalValues)
	res.AddManifestResults(results...)

	// TODO: (zeus) consider making this a runExternalValidators method similar
	// to the optional one above

	// Run external validators, if enabled.
	if entrypoints, isEnabled := validate.GetExternalValidatorEntrypoints(c.alphaSelectExternal); isEnabled {
		logger.Debugf("Running external validators: %+q", entrypoints)
		// TODO(estroz): enable timeout.
		ctx := context.Background()
		ress, err := validate.RunExternalValidators(ctx, entrypoints, c.directory)
		if err != nil {
			return nil, err
		}
		res.AddManifestResults(ress...)
	}

	return res, nil
}

// list prints a list of validators that can be turned off/on by selectors to stdout.
func (c bundleValidateCmd) list() error {
	return listOptionalValidators(os.Stdout)
}

// getBundleDataFromDir returns the bundle object and associated metadata from dir, if any.
func getBundleDataFromDir(dir string) (*apimanifests.Bundle, string, error) {
	// Gather bundle metadata.
	metadata, _, err := internalregistry.FindBundleMetadata(dir)
	if err != nil {
		return nil, "", err
	}
	manifestsDirName, hasLabel := metadata.GetManifestsDir()
	if !hasLabel {
		manifestsDirName = registrybundle.ManifestsDir
	}
	manifestsDir := filepath.Join(dir, manifestsDirName)
	// Detect mediaType.
	mediaType, err := registrybundle.GetMediaType(manifestsDir)
	if err != nil {
		return nil, "", err
	}
	// Read the bundle.
	bundle, err := apimanifests.GetBundleFromDir(manifestsDir)
	if err != nil {
		return nil, "", err
	}
	return bundle, mediaType, nil
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

// relWd returns the path of dir relative to the current working directory
func relWd(dir string) (out string, err error) {
	if out, err = filepath.Abs(dir); err != nil {
		return "", err
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Rel(wd, out)
}

// isExist returns true if path exists.
func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
