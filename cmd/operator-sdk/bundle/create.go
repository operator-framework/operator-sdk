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
	"strings"

	catalog "github.com/operator-framework/operator-sdk/internal/scaffold/olm-catalog"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type bundleCreateCmd struct {
	bundleCmd

	outputDir string
}

// newCreateCmd returns a command that will build operator bundle image or
// generate metadata for them.
func newCreateCmd() *cobra.Command {
	c := &bundleCreateCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an operator bundle image",
		Long: `The 'operator-sdk bundle create' command will build an operator
bundle image containing operator metadata and manifests, tagged with the
provided image tag.

To write metadata and a bundle image Dockerfile to disk, set '--generate-only=true'.
Bundle metadata will be generated in <directory-arg>/metadata, and a bundle Dockerfile
at <project-root>/bundle.Dockerfile. Additionally a <directory-arg>/manifests
directory will be created if one does not exist already for the specified
operator version (use --latest or --version=<semver>) This flag is useful if
you want to build an operator's bundle image manually, modify metadata before
building an image, or want to generate a 'manifests/' directory containing your
latest operator manifests for compatibility with other operator tooling.

More information on operator bundle images and metadata:
https://github.com/openshift/enhancements/blob/master/enhancements/olm/operator-bundle.md#docker

NOTE: bundle images are not runnable.
`,
		Example: `The following invocation will build a test-operator 0.1.0 bundle image using Docker.
This image will contain manifests for package channels 'stable' and 'beta':

  $ operator-sdk bundle create quay.io/example/test-operator:v0.1.0 \
      --directory ./deploy/olm-catalog/test-operator/0.1.0 \
      --package test-operator \
      --channels stable,beta \
      --default-channel stable

Assuming your operator has the same name as your operator, the tag corresponds to
a bundle directory name, and the only channel is 'stable', the above command can
be abbreviated to:

  $ operator-sdk bundle create quay.io/example/test-operator:v0.1.0 \
      --directory

The following invocation will generate test-operator bundle metadata, a manifests
dir, and Dockerfile for your latest operator version without building the image:

  $ operator-sdk bundle create \
      --generate-only \
      --directory ./deploy/olm-catalog/test-operator/0.1.0 \
      --package test-operator \
      --channels stable,beta \
      --default-channel stable
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.setDefaults(); err != nil {
				return fmt.Errorf("error setting default args: %v", err)
			}

			if err := c.validate(args); err != nil {
				return fmt.Errorf("error validating args: %v", err)
			}

			channels := strings.Join(c.channels, ",")

			if c.generateOnly {
				err := bundle.GenerateFunc(c.directory, c.outputDir, c.packageName, channels, c.defaultChannel, true)
				if err != nil {
					log.Fatal(fmt.Errorf("error generating bundle image files: %v", err))
				}
				if c.outputDir != "" {
					outputManifestsDir := filepath.Join(c.outputDir, bundle.ManifestsDir)
					if err := copyDirShallow(c.directory, outputManifestsDir); err != nil {
						return fmt.Errorf("error updating manifests dir: %v", err)
					}
				}
			} else {
				// if c.outputDir != "" {
				// 	outputManifestsDir := filepath.Join(c.outputDir, bundle.ManifestsDir)
				// 	if err := copyDirShallow(c.directory, outputManifestsDir); err != nil {
				// 		return fmt.Errorf("error updating manifests dir: %v", err)
				// 	}
				// }
				c.imageTag = args[0]
				rootDir := filepath.Dir(c.directory)
				metadataDir := filepath.Join(rootDir, bundle.MetadataDir)
				metadataDirExisted := isExist(metadataDir)
				dockerfileExisted := isExist(bundle.DockerFile)

				// Clean up transient files once the image is built, as they are no longer
				// needed.
				if !metadataDirExisted {
					defer func() {
						if err := os.RemoveAll(metadataDir); err != nil {
							log.Fatal(err)
						}
					}()
				}
				if !dockerfileExisted {
					defer func() {
						if err := os.RemoveAll(bundle.DockerFile); err != nil {
							log.Fatal(err)
						}
					}()
				}
				// for _, cleanup := range c.cleanupFuncs() {
				// 	defer cleanup()
				// }

				// Build but never overwrite existing metadata/Dockerfile.
				err := bundle.BuildFunc(c.directory, c.outputDir, c.imageTag, c.imageBuilder,
					c.packageName, channels, c.defaultChannel, false)
				if err != nil {
					log.Fatal(fmt.Errorf("error building bundle image: %v", err))
				}
			}
			return nil
		},
	}

	c.addToFlagSet(cmd.Flags())

	return cmd
}

func (c *bundleCreateCmd) addToFlagSet(fs *pflag.FlagSet) {

	fs.StringVarP(&c.directory, "directory", "d", "",
		"The directory where bundle manifests are located, ex. <project-root>/deploy/olm-catalog/test-operator/0.1.0")
	fs.StringVarP(&c.outputDir, "output-dir", "o", "",
		"Optional output directory for operator manifests")
	fs.StringVarP(&c.imageTag, "tag", "t", "",
		"The path of a registry to pull from, image name and its tag that present the bundle image "+
			"(e.g. quay.io/test/test-operator:v0.1.0)")
	fs.StringVarP(&c.packageName, "package", "p", "",
		"The name of the package that bundle image belongs to. Set if package name differs from project name")
	fs.StringSliceVarP(&c.channels, "channels", "c", []string{"stable"},
		"The list of channels that bundle image belongs to")
	fs.BoolVarP(&c.generateOnly, "generate-only", "g", false,
		"Generate metadata/, manifests/ and a Dockerfile on disk without building the bundle image")
	fs.StringVarP(&c.imageBuilder, "image-builder", "b", "docker",
		"Tool to build container images. One of: [docker, podman, buildah]")
	fs.StringVarP(&c.defaultChannel, "default-channel", "e", "",
		"The default channel for the bundle image")
}

func (c bundleCreateCmd) setDefaults() (err error) {
	projectName := filepath.Base(projutil.MustGetwd())
	if c.directory == "" {
		c.directory = filepath.Join(catalog.OLMCatalogDir, projectName)
		// Avoid discrepancy between packageName and directory if either is set
		// by only assuming the operator dir is the packageName if directory isn't set.
		c.packageName = projectName
	}
	return nil
}

func (c bundleCreateCmd) validate(args []string) error {
	if c.directory == "" {
		return fmt.Errorf("--directory must be set")
	}
	if c.packageName == "" {
		return fmt.Errorf("--package must be set")
	}
	if c.generateOnly {
		if len(args) != 0 {
			return errors.New("the command does not accept any arguments with --generate-only set")
		}
	} else {
		if len(args) != 1 {
			return errors.New("a bundle image tag is a required argument if --generate-only is not set")
		}
	}
	return nil
}

// Scenarios:
// Generate:
// - no manifests, no outputDir - create manifests normally
// - manifests, no outputDir - do not create manifests
// - no manifests, outputDir - create manifests in outputDir
// - manifests, outputDir - create manifests in outputDir
func (c bundleCreateCmd) runGenerate() (err error) {

	return nil
}

// Build:
// - no manifests, no metadata, no outputDir
// - no manifests, metadata, no outputDir
// - manifests, no metadata, no outputDir
// - no manifests, no metadata, outputDir
// - no manifests, metadata, outputDir
// - manifests, no metadata, outputDir
func (c bundleCreateCmd) runBuild() (err error) {

	return nil
}

func copyDirShallow(from, to string) error {
	infos, err := ioutil.ReadDir(from)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(to, fileutil.DefaultDirFileMode); err != nil {
		return err
	}

	for _, info := range infos {
		fromPath := filepath.Join(from, info.Name())
		toPath := filepath.Join(to, info.Name())
		if !info.IsDir() {
			b, err := ioutil.ReadFile(fromPath)
			if err != nil {
				return err
			}
			data := "(empty)"
			if len(b) > 20 {
				data = string(b[:20])
			}
			fmt.Printf("writing %s to %s: %s\n", fromPath, toPath, data)
			if err = ioutil.WriteFile(toPath, b, fileutil.DefaultFileMode); err != nil {
				return err
			}
		} else {
			log.Debugf("Skipping copy %s to %s", fromPath, toPath)
		}
	}

	return nil
}

// // cleanupFuncs returns a set of funcs to clean up after 'bundle create'.
// func (c bundleCreateCmd) cleanupFuncs() (fs []func()) {
// 	// If output-dir is set we don't want to remove files, since the user has
// 	// specified they want a directory generated.
// 	metaDir := filepath.Join(c.directory, bundle.MetadataDir)
// 	metaExists := isExist(metaDir)
// 	dockerFileExists := isExist(bundle.DockerFile)
// 	fs = append(fs,
// 		func() {
// 			if !metaExists {
// 				if err := os.RemoveAll(metaDir); err != nil {
// 					log.Fatal(err)
// 				}
// 			}
// 		},
// 		func() {
// 			if !dockerFileExists {
// 				if err := os.RemoveAll(bundle.DockerFile); err != nil {
// 					log.Fatal(err)
// 				}
// 			}
// 		})
// 	return fs
// }

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
