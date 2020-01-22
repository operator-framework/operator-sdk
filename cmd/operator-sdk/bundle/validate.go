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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-registry/pkg/lib/bundle"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// newValidateCmd returns a command that will validate an operator bundle image.
func newValidateCmd() *cobra.Command {
	c := bundleCmd{}
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate an operator bundle image",
		Long: `The 'operator-sdk bundle validate' command will validate both content and
format of an operator bundle image containing operator metadata and manifests.
This command will exit with a non-zero exit code if any validation tests fail.

Note: the image being validated must exist in a remote registry, not just locally.`,
		Example: `The following command flow will generate test-operator bundle image manifests
and validate that image:

$ cd ${HOME}/go/test-operator

# Generate manifests locally.
$ operator-sdk bundle build --generate-only

# Modify the metadata and Dockerfile.
$ cd ./deploy/olm-catalog/test-operator
$ vim ./metadata/annotations.yaml
$ vim ./Dockerfile

# Build and push the image using the docker CLI.
$ docker build -t quay.io/example/test-operator:v0.1.0 .
$ docker push quay.io/example/test-operator:v0.1.0

# Ensure the image with modified metadata/Dockerfile is valid.
$ operator-sdk bundle validate quay.io/example/test-operator:v0.1.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("a bundle image tag is a required argument, ex. example.com/test-operator:v0.1.0")
			}
			c.imageTag = args[0]

			dir, err := ioutil.TempDir("", "bundle-")
			if err != nil {
				log.Fatal(err)
			}
			defer func() {
				if err = os.RemoveAll(dir); err != nil {
					log.Error(err.Error())
				}
			}()
			logger := log.WithFields(log.Fields{
				"container-tool": c.imageBuilder,
				"bundle-dir":     dir,
			})
			log.SetLevel(log.DebugLevel)
			val := bundle.NewImageValidator(c.imageBuilder, logger)
			if err = val.PullBundleImage(c.imageTag, dir); err != nil {
				log.Fatalf("Error to unpacking image: %v", err)
			}

			log.Info("Validating bundle image format and contents")

			if err = val.ValidateBundleFormat(dir); err != nil {
				log.Fatalf("Bundle format validation failed: %v", err)
			}
			manifestsDir := filepath.Join(dir, bundle.ManifestsDir)
			if err = val.ValidateBundleContent(manifestsDir); err != nil {
				log.Fatalf("Bundle content validation failed: %v", err)
			}

			log.Info("All validation tests have completed successfully")

			return nil
		},
	}

	cmd.Flags().StringVarP(&c.imageBuilder, "image-builder", "b", "docker",
		"Tool to extract container images. One of: [docker, podman]")

	return cmd
}
