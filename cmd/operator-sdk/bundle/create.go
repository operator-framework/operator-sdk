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

	catalog "github.com/operator-framework/operator-sdk/internal/generate/olm-catalog"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/blang/semver"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	scorecard "github.com/operator-framework/operator-sdk/internal/scorecard/alpha"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type bundleCreateCmd struct {
	bundleCmd

	outputDir string
	overwrite bool
}

// newCreateCmd returns a command that will build operator bundle image or
// generate metadata for them.
//nolint:lll
func newCreateCmd() *cobra.Command {
	c := &bundleCreateCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an operator bundle image",
		Long: `The 'operator-sdk bundle create' command will build an operator
bundle image containing operator metadata and manifests, tagged with the
provided image tag.

To write all files required to build a bundle image without building the
image, set '--generate-only=true'. A bundle.Dockerfile and bundle metadata
will be written if '--generate-only=true':

` + "```" + `
  $ operator-sdk bundle create --generate-only --directory ./deploy/olm-catalog/test-operator/manifests
  $ ls .
  ...
  bundle.Dockerfile
  ...
  $ tree ./deploy/olm-catalog/test-operator/
  ./deploy/olm-catalog/test-operator/
  ├── manifests
  │   ├── example.com_tests_crd.yaml
  │   └── test-operator.clusterserviceversion.yaml
  └── metadata
      └── annotations.yaml
` + "```" + `

'--generate-only' is useful if you want to build an operator's bundle image
manually or modify metadata before building an image.

More information about operator bundles and metadata:
https://github.com/operator-framework/operator-registry#manifest-format.

NOTE: bundle images are not runnable.
`,
		Example: `The following invocation will build a test-operator 0.1.0 bundle image using Docker.
This image will contain manifests for package channels 'stable' and 'beta':

  $ operator-sdk bundle create quay.io/example/test-operator:v0.1.0 \
      --directory ./deploy/olm-catalog/test-operator/manifests \
      --package test-operator \
      --channels stable,beta \
      --default-channel stable

Assuming your operator has the same name as your repo directory and the only
channel is 'stable', the above command can be abbreviated to:

  $ operator-sdk bundle create quay.io/example/test-operator:v0.1.0

The following invocation will generate test-operator bundle metadata and a
bundle.Dockerfile for your latest operator version without building the image:

  $ operator-sdk bundle create \
      --generate-only \
      --package test-operator \
      --channels beta \
      --default-channel beta
`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			if err = c.setDefaults(); err != nil {
				log.Fatalf("Failed to default args: %v", err)
			}

			if err = c.validate(args); err != nil {
				return fmt.Errorf("error validating args: %v", err)
			}

			if c.generateOnly {
				err = c.runGenerate()
			} else {
				c.imageTag = args[0]
				err = c.runBuild()
			}

			if err != nil {
				log.Fatal(err)
			}

			return nil
		},
	}

	c.addToFlagSet(cmd.Flags())
	return cmd
}

func (c *bundleCreateCmd) addToFlagSet(fs *pflag.FlagSet) {
	fs.StringVarP(&c.directory, "directory", "d", "",
		"The directory where bundle manifests are located, ex. <project-root>/deploy/olm-catalog/test-operator/manifests")
	fs.StringVarP(&c.outputDir, "output-dir", "o", "",
		"Optional output directory for operator manifests")
	fs.StringVarP(&c.imageTag, "tag", "t", "",
		"The path of a registry to pull from, image name and its tag that present the bundle image "+
			"(e.g. quay.io/test/test-operator:v0.1.0)")
	fs.StringVarP(&c.packageName, "package", "p", "",
		"The name of the package that bundle image belongs to. Set if package name differs from project name")
	fs.StringVarP(&c.channels, "channels", "c", "stable",
		"The comma-separated list of channels that bundle image belongs to")
	fs.BoolVarP(&c.generateOnly, "generate-only", "g", false,
		"Generate metadata/, manifests/ and a Dockerfile on disk without building the bundle image")
	fs.BoolVar(&c.overwrite, "overwrite", false,
		"Overwrite bundle.Dockerfile, manifests and metadata dirs if they exist. "+
			"If --output-dir is also set, the original files will not be overwritten")
	fs.StringVarP(&c.imageBuilder, "image-builder", "b", "docker",
		"Tool to build container images. One of: [docker, podman, buildah]")
	fs.StringVarP(&c.defaultChannel, "default-channel", "e", "",
		"The default channel for the bundle image")
}

func (c *bundleCreateCmd) setDefaults() (err error) {
	if c.packageName == "" {
		c.packageName = filepath.Base(projutil.MustGetwd())
	}
	defaultManifestsDir := filepath.Join(catalog.OLMCatalogDir, c.packageName, bundle.ManifestsDir)
	if c.directory == "" {
		if isNotExist(defaultManifestsDir) {
			return fmt.Errorf("default manifests directory %s does not exist; "+
				"set --directory to a valid bundle manifests directory", defaultManifestsDir)
		}
		c.directory = defaultManifestsDir
	}

	// Clean and make paths relative for less verbose error messages. Don't return
	// an error if we cannot.
	if dir, err := relWd(c.directory); err == nil {
		c.directory = dir
	}

	// Ensure a default channel is present if there only one channel. Don't infer
	// default otherwise; the user must set this value.
	if c.defaultChannel == "" && strings.Count(c.channels, ",") == 0 {
		c.defaultChannel = c.channels
	}

	return nil
}

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

func (c bundleCreateCmd) validate(args []string) error {
	if c.directory == "" {
		return fmt.Errorf("--directory must be set")
	}
	if c.packageName == "" {
		return fmt.Errorf("--package must be set")
	}
	// Bundle commands only work with bundle directory formats, not package
	// manifests formats.
	if isPackageManifestsDir(c.directory, c.packageName) {
		return fmt.Errorf("bundle commands can only be used on bundle directory formats")
	}
	if c.generateOnly {
		if len(args) != 0 {
			return errors.New("the command does not accept any arguments if --generate-only=true")
		}
	} else {
		if len(args) != 1 {
			return errors.New("a bundle image tag is a required argument if --generate-only=true")
		}
	}
	return nil
}

// runGenerate generates a bundle.Dockerfile, and manifests/ and metadata/ dirs with scorecard config
// copied to the bundle image.
func (c bundleCreateCmd) runGenerate() error {
	if c.generateOnly {
		c.overwrite = true
	}

	err := bundle.GenerateFunc(c.directory, c.outputDir, c.packageName, c.channels, c.defaultChannel, c.overwrite)
	if err != nil {
		return fmt.Errorf("error generating bundle image files: %v", err)
	}
	if err = copyScorecardConfig(); err != nil {
		return err
	}
	return nil
}

// runBuild runs the equivalent of runGenerate then builds a bundle image. If
// a manifest/, metadata/ or bundle.DockerFile do not exist, they are removed.
func (c bundleCreateCmd) runBuild() error {
	rootDir := filepath.Dir(c.directory)
	metadataDir := filepath.Join(rootDir, bundle.MetadataDir)

	// Clean up transient files once the image is built, as they are no longer
	// needed.
	if isNotExist(metadataDir) {
		defer remove(metadataDir)
	}
	if isNotExist(bundle.DockerFile) {
		defer remove(bundle.DockerFile)
	}

	// Build with overwrite-able option.
	err := c.buildFunc()
	if err != nil {
		return fmt.Errorf("error building bundle image: %v", err)
	}
	return nil
}

// buildFunc is used to build a container image from a list of manifests and generates dockerfile and annotations.yaml.
func (c bundleCreateCmd) buildFunc() error {
	_, err := os.Stat(c.directory)
	if os.IsNotExist(err) {
		return err
	}

	err = c.runGenerate()
	if err != nil {
		return err
	}

	// Build bundle image
	log.Info("Building bundle image")
	buildCmd, err := bundle.BuildBundleImage(c.imageTag, c.imageBuilder)
	if err != nil {
		return err
	}

	if err := bundle.ExecuteCommand(buildCmd); err != nil {
		return err
	}

	return nil
}

// remove removes path from disk. Used in defer statements.
func remove(path string) {
	if err := os.RemoveAll(path); err != nil {
		log.Fatal(err)
	}
}

// isExist returns true if path exists.
func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

// isNotExist returns true if path does not exist.
func isNotExist(path string) bool {
	_, err := os.Stat(path)
	return err != nil && os.IsNotExist(err)
}

// isPackageManifestsDir checks if dir is a package manifests format directory
// by checking for the existence of a package manifest and a semver-named directory.
func isPackageManifestsDir(dir, operatorName string) bool {
	packageManifestPath := filepath.Join(filepath.Dir(dir), operatorName+".package.yaml")
	_, err := semver.ParseTolerant(filepath.Clean(filepath.Base(dir)))
	return isExist(packageManifestPath) && err == nil
}

// copyScorecardConfigToBundle checks if bundle.Dockerfile and scorecard config exists in
// the operator project. If it does, it injects the scorecard configuration into bundle
// image.
// TODO: Add labels to annotations.yaml and bundle.dockerfile.
func copyScorecardConfig() error {
	if isExist(bundle.DockerFile) && isExist(scorecard.ConfigDirName) {
		scorecardFileContent := fmt.Sprintf("COPY %s %s\n", scorecard.ConfigDirName, scorecard.ConfigDirPath)
		err := projutil.RewriteFileContents(bundle.DockerFile, "COPY", scorecardFileContent)
		if err != nil {
			return fmt.Errorf("error rewriting dockerfile, %v", err)
		}
	}
	return nil
}
