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

	"github.com/blang/semver"
	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type bundleCreateCmd struct {
	bundleCmd

	outputDir        string
	version          string
	useLatestVersion bool
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
      --version 0.1.0 \
      --directory ./deploy/olm-catalog/test-operator \
      --package test-operator \
      --channels stable,beta \
      --default-channel stable

Assuming your operator has the same name as your operator, the tag corresponds to
a bundle directory name, and the only channel is 'stable', the above command can
be abbreviated to:

  $ operator-sdk bundle create quay.io/example/test-operator:v0.1.0 --version 0.1.0

The following invocation will generate test-operator bundle metadata, a manifests
dir, and Dockerfile for your latest operator version without building the image:

  $ operator-sdk bundle create \
      --generate-only \
      --latest \
      --directory ./deploy/olm-catalog/test-operator \
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

			if !c.generateOnly {
				c.imageTag = args[0]
			}
			if err := c.run(); err != nil {
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
		"The directory where bundle manifests are located, ex. <project-root>/deploy/olm-catalog/<operator-name>. "+
			"Set if package name differs from project name")
	fs.StringVarP(&c.outputDir, "output-dir", "o", "",
		"Optional output directory for operator manifests")
	fs.StringVarP(&c.version, "version", "v", "",
		"Version of operator to build an image for. Must match a directory name of a bundle dir. "+
			"Set this if you do not have a 'manifests' directory at <project-root>/deploy/olm-catalog/<operator-name>/manifests")
	fs.BoolVar(&c.useLatestVersion, "latest", false,
		"Use the latest semantically versioned directory in <project-root>/deploy/olm-catalog/<operator-name>")
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
	// Validate semver if not latest
	if c.version != "" {
		if c.useLatestVersion {
			return fmt.Errorf("cannot set both --latest and --version")
		}
		if _, err := semver.Parse(c.version); err != nil {
			return fmt.Errorf("version %s is invalid: %v", c.version, err)
		}
	}
	return nil
}

func (c bundleCreateCmd) run() (err error) {
	channels := strings.Join(c.channels, ",")
	manifestsDir := filepath.Join(c.directory, bundle.ManifestsDir)
	manifestsDirExisted := isExist(manifestsDir)

	// If the latest version is passed, find the highest semver in c.directory.
	if c.useLatestVersion {
		if c.version, err = findLatestSemverDir(c.directory); err != nil {
			return fmt.Errorf("error finding latest operator bundle dir: %v", err)
		}
	}

	// version will be empty if neither useLatestVersion nor version were set
	// by the user, so they want manifests/ left alone. Otherwise update it.
	if c.version != "" {
		versionedDir := filepath.Join(c.directory, c.version)
		if err := copyDirShallow(versionedDir, manifestsDir); err != nil {
			return fmt.Errorf("error updating manifests dir: %v", err)
		}
	}

	if c.generateOnly {
		err := bundle.GenerateFunc(manifestsDir, c.outputDir, c.packageName, channels, c.defaultChannel, true)
		if err != nil {
			return fmt.Errorf("error generating bundle image files: %v", err)
		}
		return nil
	}

	// Clean up transient files once the image is built, as they are no longer
	// needed.
	if !manifestsDirExisted {
		defer func() {
			if err := os.RemoveAll(manifestsDir); err != nil {
				log.Fatal(err)
			}
		}()
	}
	for _, cleanup := range c.cleanupFuncs() {
		defer cleanup()
	}

	// Build but never overwrite existing metadata/Dockerfile.
	err = bundle.BuildFunc(manifestsDir, c.outputDir, c.imageTag, c.imageBuilder,
		c.packageName, channels, c.defaultChannel, false)
	if err != nil {
		return fmt.Errorf("error building bundle image: %v", err)
	}

	return nil
}

func findLatestSemverDir(operatorDir string) (latestVerStr string, err error) {
	infos, err := ioutil.ReadDir(operatorDir)
	if err != nil {
		return "", err
	}
	versions := semver.Versions{}
	for _, info := range infos {
		if info.IsDir() {
			childDir := filepath.Clean(info.Name())
			ver, err := semver.Parse(childDir)
			if err != nil {
				log.Debugf("Skipping non-semver dir %s: %v", childDir, err)
				continue
			}
			versions = append(versions, ver)
		}
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no semver dirs found in %s", operatorDir)
	}
	semver.Sort(versions)
	latestVerStr = versions[len(versions)-1].String()
	return latestVerStr, nil
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
			if err = ioutil.WriteFile(toPath, b, info.Mode()); err != nil {
				return err
			}
		} else {
			log.Debugf("Skipping copy %s to %s", fromPath, toPath)
		}
	}

	return nil
}

// cleanupFuncs returns a set of funcs to clean up after 'bundle create'.
func (c bundleCreateCmd) cleanupFuncs() (fs []func()) {
	rootDir := c.outputDir
	if rootDir == "" {
		rootDir = c.directory
	}

	metaDir := filepath.Join(rootDir, bundle.MetadataDir)
	metaExists := isExist(metaDir)
	dockerFileExists := isExist(bundle.DockerFile)
	fs = append(fs,
		func() {
			if !metaExists {
				if err := os.RemoveAll(metaDir); err != nil {
					log.Fatal(err)
				}
			}
		},
		func() {
			if !dockerFileExists {
				if err := os.RemoveAll(bundle.DockerFile); err != nil {
					log.Fatal(err)
				}
			}
		})
	return fs
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}
