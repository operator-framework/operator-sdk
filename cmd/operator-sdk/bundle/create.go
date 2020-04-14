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
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	catalog "github.com/operator-framework/operator-sdk/internal/generate/olm-catalog"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
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
func newCreateCmd() *cobra.Command {
	c := &bundleCreateCmd{}

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an operator bundle image",
		Long: `The 'operator-sdk bundle create' command will build an operator
bundle image containing operator metadata and manifests, tagged with the
provided image tag.

To write all files required to build a bundle image without building the
image, set '--generate-only=true'. A bundle Dockerfile, bundle metadata, and
a 'manifests/' directory containing your bundle manifests will be written if
'--generate-only=true':

	$ operator-sdk bundle create --generate-only --directory ./deploy/olm-catalog/test-operator/0.1.0
	$ ls .
	...
	bundle.Dockerfile
	...
	$ tree ./deploy/olm-catalog/test-operator/
	└── 0.1.0
		└── example.com_tests_crd.yaml
		└── test-operator.v0.1.0.clusterserviceversion.yaml
	└── manifests
		└── example.com_tests_crd.yaml
		└── test-operator.v0.1.0.clusterserviceversion.yaml
	└── metadata
		└── annotations.yaml

'--generate-only' is useful if you want to build an operator's bundle image
manually, modify metadata before building an image, or want to generate a
'manifests/' directory containing your operator manifests for compatibility
with other operator tooling.

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

Assuming your operator has the same name as your repo directory and the only
channel is 'stable', the above command can be abbreviated to:

  $ operator-sdk bundle create quay.io/example/test-operator:v0.1.0 \
      --directory ./deploy/olm-catalog/test-operator/0.1.0

The following invocation will generate test-operator bundle metadata, a
'manifests/' dir, and Dockerfile for your latest operator version without
building the image:

  $ operator-sdk bundle create \
      --generate-only \
      --directory ./deploy/olm-catalog/test-operator/0.1.0 \
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

			rootDir := filepath.Dir(c.directory)
			manifestsDir := filepath.Join(rootDir, bundle.ManifestsDir)

			// To ensure users don't accidentally overwrite their manifests dir
			// created previously, make sure they have set --overwrite or no
			// contents of the directory differ in the source directory.
			if !c.overwrite && isExist(manifestsDir) && c.outputDir == filepath.Dir(c.directory) {
				dirsDiffer, err := areDirsDiff(c.directory, manifestsDir)
				if err != nil {
					log.Fatal(err)
				}
				if dirsDiffer {
					log.Fatalf("'manifests' dir already exists at %s. Set --overwrite=true "+
						"to overwrite its contents with contents in %s",
						manifestsDir, c.directory)
				}
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
		"The directory where bundle manifests are located, ex. <project-root>/deploy/olm-catalog/test-operator/0.1.0")
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
	if c.directory == "" {
		c.directory = filepath.Join(catalog.OLMCatalogDir, c.packageName, bundle.ManifestsDir)
	}

	// Clean and make paths relative for less verbose error messages.
	if c.directory, err = relDir(c.directory); err != nil {
		return err
	}
	// Set outputDir in any case so we make the operator-registry file generator
	// write 'manifests/' every time, and handle cleanup logic in runBuild().
	if c.outputDir == "" {
		c.outputDir = filepath.Dir(c.directory)
	}
	if c.outputDir, err = relDir(c.outputDir); err != nil {
		return err
	}

	return nil
}

func relDir(dir string) (out string, err error) {
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

// runGenerate generates a bundle.Dockerfile, and manifests/ and metadata/ dirs,
// always overwriting their contents.
func (c bundleCreateCmd) runGenerate() error {
	err := bundle.GenerateFunc(c.directory, c.outputDir, c.packageName, c.channels, c.defaultChannel, true)
	if err != nil {
		return fmt.Errorf("error generating bundle image files: %v", err)
	}
	return nil
}

// runBuild runs the equivalent of runGenerate then builds a bundle image. If
// a manifest/, metadata/ or bundle.DockerFile do not exist, they are removed.
func (c bundleCreateCmd) runBuild() error {
	rootDir := filepath.Dir(c.directory)
	metadataDir := filepath.Join(rootDir, bundle.MetadataDir)
	manifestsDir := filepath.Join(rootDir, bundle.ManifestsDir)

	// Clean up transient files once the image is built, as they are no longer
	// needed.
	if !isExist(manifestsDir) {
		defer remove(manifestsDir)
	}
	if !isExist(metadataDir) {
		defer remove(metadataDir)
	}
	if !isExist(bundle.DockerFile) {
		defer remove(bundle.DockerFile)
	}

	// Build with overwrite-able option.
	err := bundle.BuildFunc(c.directory, c.outputDir, c.imageTag, c.imageBuilder,
		c.packageName, c.channels, c.defaultChannel, c.overwrite)
	if err != nil {
		return fmt.Errorf("error building bundle image: %v", err)
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

// areDirsDiff returns true if either file names or file contents differ
// between dirA and dirB.
func areDirsDiff(dirA, dirB string) (bool, error) {
	if filepath.Clean(dirA) == filepath.Clean(dirB) {
		return false, nil
	}
	fileMapA, err := getDirFileMap(dirA)
	if err != nil {
		return false, err
	}
	fileMapB, err := getDirFileMap(dirB)
	if err != nil {
		return false, err
	}
	if len(fileMapB) != len(fileMapA) {
		return true, nil
	}
	for pathA, contentsA := range fileMapA {
		contentsB, hasPathA := fileMapB[pathA]
		if !hasPathA || contentsA != contentsB {
			return true, nil
		}
	}
	return false, nil
}

// getDirFileMap returns a map of file names to contents as strings for all
// normal files in dir (recursive).
func getDirFileMap(dir string) (map[string]string, error) {
	fileMap := make(map[string]string)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			fileMap[filepath.Base(path)] = hashContents(b)
		}
		return nil
	})
	return fileMap, err
}

// hashContents returns the hexadecimal representation of hashed b.
func hashContents(b []byte) string {
	h := sha256.New()
	_, _ = h.Write(b)
	return fmt.Sprintf("%x", h.Sum(nil))
}
