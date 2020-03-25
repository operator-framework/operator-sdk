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
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/operator-framework/operator-sdk/internal/flags"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	// Version of operator-registry (read: opm) behing used.
	registryVersion = "v1.6.0"
	// Path to opm program.
	opmCmdPath = "github.com/operator-framework/operator-registry/cmd/opm"
)

// bundleIndexCmd configures 'opm index' subcommand invocation.
type bundleIndexCmd struct {
	bundlesImages      string
	fromIndex, toIndex string
	dockerfileName     string
	imageBuilder       string
	generateOnly       bool
	permissive         bool
}

// newAddCmd returns a command that will add an operator bundle image to an
// operator index image (catalog).
func newAddCmd() *cobra.Command {
	c := &bundleIndexCmd{}
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add an operator bundle image to an operator index image",
		Long: fmt.Sprintf(`The 'operator-sdk bundle add' command will add an operator bundle image to an
existing operator index image, or create an index image.

This command downloads and shells out to the 'opm' binary under the hood. The
version downloaded by 'bundle add' is: %s

Bundle images being passed to 'bundle add' must be present remotely, and access
to the remote repository should be enabled in the command line environment.

More information on operator index images:
https://github.com/openshift/enhancements/blob/master/enhancements/olm/operator-registry.md
More information on 'opm':
https://github.com/operator-framework/operator-registry/blob/master/docs/design/opm-tooling.md
`,
			registryVersion),
		Example: `The following invocation will create a new test-operator bundle index image:

  $ operator-sdk bundle add quay.io/example/test-operator:v0.1.0 \
      --to-index quay.io/example/test-operator-index:v0.1.0

The following invocation will add a test-operator bundle image to an existing
index image at version v0.1.0, creating a new index image at version v0.2.0:

  $ operator-sdk bundle add quay.io/example/test-operator:v0.2.0 \
      --from-index quay.io/example/test-operator-index:v0.1.0 \
      --to-index quay.io/example/test-operator-index:v0.2.0
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return errors.New("a comma-separated list of bundle image tags is a required argument, " +
					"ex. 'example.com/test-operator:v0.1.0,example.com/test-operator:v0.2.0'")
			}
			c.bundlesImages = args[0]

			if err := c.validate(); err != nil {
				return fmt.Errorf("error validating args: %v", err)
			}

			binaryPath := filepath.Join("bin", "opm")
			if runtime.GOOS == "windows" {
				binaryPath += ".exe"
			}
			if err := c.buildOPM(binaryPath); err != nil {
				log.Fatalf("Error building image builder: %v", err)
			}

			// Clean up database and index Dockerfile once the image is built,
			// as they are no longer needed.
			for _, cleanup := range c.cleanupFuncs() {
				defer cleanup()
			}

			if err := c.runOPMIndexAdd(binaryPath); err != nil {
				log.Fatalf("Error building index image: %v", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&c.fromIndex, "from-index", "f", "", "Previous index to build new index image from")
	cmd.Flags().StringVarP(&c.toIndex, "to-index", "t", "", "Tag for new index image being built")
	if err := cmd.MarkFlagRequired("to-index"); err != nil {
		panic(err)
	}
	cmd.Flags().StringVar(&c.dockerfileName, "dockerfile-name", "",
		"Name of the Dockerfile to generate if --generate-only is set. Default is 'Dockerfile'")
	cmd.Flags().StringVar(&c.imageBuilder, "image-builder", "docker",
		"Tool to build container images. One of: [docker, podman]")
	cmd.Flags().BoolVarP(&c.generateOnly, "generate-only", "g", false,
		"Generate the underlying database and a Dockerfile without building the index container image")
	cmd.Flags().BoolVar(&c.permissive, "permissive", false,
		"Allow registry load errors without exiting the build")

	return cmd
}

func (c bundleIndexCmd) validate() error {
	if !c.generateOnly && c.dockerfileName != "" {
		return fmt.Errorf("dockerfile name can only be set if generating files")
	}
	return nil
}

// cleanupFuncs returns a set of general funcs to clean up after a bundle
// subcommand.
func (c bundleIndexCmd) cleanupFuncs() (fs []func()) {
	databaseDir := "database"
	dockerFile := "index.DockerFile"
	databaseExists := isExist(databaseDir)
	dockerFileExists := isExist(dockerFile)
	fs = append(fs,
		func() {
			if !databaseExists {
				_ = os.RemoveAll(databaseDir)
			}
		},
		func() {
			if !dockerFileExists {
				_ = os.RemoveAll(dockerFile)
			}
		})
	return fs
}

// buildOPM creates and download 'opm' to a binary directory using 'go get' to
// download a particular version, specified by registryVersion, to install.
// TODO(estroz): shell out to make after kubebuilder integration.
func (c bundleIndexCmd) buildOPM(binaryPath string) error {
	binaryDir := filepath.Dir(binaryPath)
	if err := os.MkdirAll(binaryDir, fileutil.DefaultDirFileMode); err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(binaryDir); err != nil {
		return err
	}
	defer func() {
		if err := os.Chdir(wd); err != nil {
			log.Fatal(err)
		}
	}()
	defer func() {
		if err := os.RemoveAll("go.mod"); err != nil {
			log.Fatal(err)
		}
	}()
	defer func() {
		if err := os.RemoveAll("go.sum"); err != nil {
			log.Fatal(err)
		}
	}()

	cmds := []*exec.Cmd{
		exec.Command("go", "mod", "init"),
		exec.Command("go", "get", fmt.Sprintf("%s@%s", opmCmdPath, registryVersion)),
		exec.Command("go", "build", "-o", filepath.Base(binaryPath), opmCmdPath),
	}
	for _, cmd := range cmds {
		if err := projutil.ExecCmd(cmd); err != nil {
			return err
		}
	}
	return nil
}

// runOPMIndexAdd runs 'opm index add', setting flags from values in c. No
// files are generated unless generateOnly is true.
// Note: operator-registry sports a Go library to handle index building, but
// it requires operator-sdk be built with CGO_ENABLED=1. This is currently
// not possible due to k8s dependency requirements.
func (c bundleIndexCmd) runOPMIndexAdd(binaryPath string) error {
	// Construct 'opm' args.
	args := []string{
		"index", "add",
		"--bundles", c.bundlesImages,
		"--tag", c.toIndex,
		// Set these with = otherwise cobra things they're always true.
		"--generate=" + strconv.FormatBool(c.generateOnly),
		"--permissive=" + strconv.FormatBool(c.permissive),
		"--debug=" + strconv.FormatBool(viper.GetBool(flags.VerboseOpt)),
	}
	switch {
	case c.fromIndex != "":
		args = append(args, "--from-index", c.fromIndex)
		fallthrough
	case c.dockerfileName != "":
		args = append(args, "--out-dockerfile", c.dockerfileName)
		fallthrough
	case c.imageBuilder != "":
		args = append(args, "--container-tool", c.imageBuilder)
	}
	cmd := exec.Command(binaryPath, args...)

	log.Infof("Building index image %s", c.toIndex)
	if err := projutil.ExecCmd(cmd); err != nil {
		return err
	}
	return nil
}
