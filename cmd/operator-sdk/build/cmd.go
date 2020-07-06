// Copyright 2018 The Operator-SDK Authors
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

package build

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	kbutil "github.com/operator-framework/operator-sdk/internal/util/kubebuilder"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	"github.com/google/shlex"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type buildCmd struct {
	image               string
	imageBuildArgs      string
	imageBuilder        string
	splitImageBuildArgs []string

	// todo: remove when the legacy layout is no longer supported
	// Deprecated
	goBuildArgs string
}

func NewCmd() *cobra.Command {
	c := &buildCmd{}
	buildCmd := &cobra.Command{
		Use:   "build <image>",
		Short: "Compiles code and builds artifacts",
		Long: `The operator-sdk build command compiles the Operator code into an executable binary
and generates the Dockerfile manifest.

'< image >' is the container image to be built, e.g. "quay.io/example/operator:v0.0.1".
This image will be automatically set in the deployment manifests.

After build completes, the image would be built locally in docker. Then it needs to
be pushed to remote registry.
For example:

	$ operator-sdk build quay.io/example/operator:v0.0.1
	$ docker push quay.io/example/operator:v0.0.1
`,
		PreRunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.validate(args)
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			return c.run()
		},
	}
	buildCmd.Flags().StringVar(&c.imageBuildArgs, "image-build-args", "",
		"Extra image build arguments as one string such as \"--build-arg https_proxy=$https_proxy\"")
	buildCmd.Flags().StringVar(&c.imageBuilder, "image-builder", "docker",
		"Tool to build OCI images. One of: [docker, podman, buildah]")

	// todo: remove when the legacy layout is no longer supported
	if !kbutil.HasProjectFile() {
		buildCmd.Flags().StringVar(&c.goBuildArgs, "go-build-args", "",
			"Extra Go build arguments as one string such as \"-ldflags -X=main.xyz=abc\"")
	}
	return buildCmd
}

func (c *buildCmd) validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("an image name is required")
	}
	c.image = args[0]

	var err error
	if c.imageBuildArgs != "" {
		c.splitImageBuildArgs, err = shlex.Split(c.imageBuildArgs)
		if err != nil {
			return fmt.Errorf("image-build-args is not parseable: %v", err)
		}
	}
	for i, v := range c.splitImageBuildArgs {
		fmt.Println(i, v)
	}

	return nil
}

func (c *buildCmd) run() error {
	projutil.MustInProjectRoot()

	if kbutil.HasProjectFile() {
		if err := c.doImageBuild("Dockerfile"); err != nil {
			log.Fatalf("Failed to build image %s: %v", c.image, err)
		}
		return nil
	}

	// todo: remove when the legacy layout is no longer supported
	// note that the above if will no longer be required as well.
	if err := c.doLegacyBuild(); err != nil {
		log.Fatalf("Failed to build image %s: %v", c.image, err)
	}
	return nil
}

// doImageBuild will execute the build command for the given Dockerfile path and image
func (c *buildCmd) doImageBuild(dockerFilePath string) error {
	log.Infof("Building OCI image %s", c.image)
	buildCmd, err := c.createBuildCommand(c.imageBuilder, ".", dockerFilePath)
	if err != nil {
		return err
	}
	if err := projutil.ExecCmd(buildCmd); err != nil {
		return err
	}
	log.Info("Operator build complete.")
	return nil
}

func (c *buildCmd) createBuildCommand(imageBuilder, context, dockerFile string) (*exec.Cmd, error) {
	var args []string
	switch imageBuilder {
	case "docker", "podman":
		args = append(args, "build", "-f", dockerFile, "-t", c.image)
	case "buildah":
		args = append(args, "bud", "--format=docker", "-f", dockerFile, "-t", c.image)
	default:
		return nil, fmt.Errorf("%s is not supported image builder", imageBuilder)
	}

	args = append(args, c.splitImageBuildArgs...)
	args = append(args, context)

	return exec.Command(imageBuilder, args...), nil
}

// todo: remove when the legacy layout is no longer supported
// Deprecated: Used just for the legacy layout
// --
// doLegacyBuild will build projects with the legacy layout.
func (c *buildCmd) doLegacyBuild() error {
	goBuildEnv := append(os.Environ(), "GOOS=linux")
	// If CGO_ENABLED is not set, set it to '0'.
	if _, ok := os.LookupEnv("CGO_ENABLED"); !ok {
		goBuildEnv = append(goBuildEnv, "CGO_ENABLED=0")
	}
	absProjectPath := projutil.MustGetwd()
	projectName := filepath.Base(absProjectPath)

	// Don't need to build Go code if a non-Go Operator.
	if projutil.IsOperatorGo() {
		trimPath := fmt.Sprintf("all=-trimpath=%s", filepath.Dir(absProjectPath))
		args := []string{"-gcflags", trimPath, "-asmflags", trimPath}

		if c.goBuildArgs != "" {
			splitArgs := strings.Fields(c.goBuildArgs)
			args = append(args, splitArgs...)
		}

		opts := projutil.GoCmdOptions{
			BinName:     filepath.Join(absProjectPath, scaffold.BuildBinDir, projectName),
			PackagePath: path.Join(projutil.GetGoPkg(), filepath.ToSlash(scaffold.ManagerDir)),
			Args:        args,
			Env:         goBuildEnv,
		}
		if err := projutil.GoBuild(opts); err != nil {
			log.Fatalf("Failed to build operator binary: %v", err)
		}
	}
	return c.doImageBuild("build/Dockerfile")
}
