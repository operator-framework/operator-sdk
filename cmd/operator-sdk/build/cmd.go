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

var (
	imageBuildArgs string
	imageBuilder   string

	// todo: remove when the legacy layout is no longer supported
	// Deprecated
	goBuildArgs string
)

func NewCmd() *cobra.Command {
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
		RunE: buildFunc,
	}
	buildCmd.Flags().StringVar(&imageBuildArgs, "image-build-args", "",
		"Extra image build arguments as one string such as \"--build-arg https_proxy=$https_proxy\"")
	buildCmd.Flags().StringVar(&imageBuilder, "image-builder", "docker",
		"Tool to build OCI images. One of: [docker, podman, buildah]")

	// todo: remove when the legacy layout is no longer supported
	if !kbutil.HasProjectFile() {
		buildCmd.Flags().StringVar(&goBuildArgs, "go-build-args", "",
			"Extra Go build arguments as one string such as \"-ldflags -X=main.xyz=abc\"")
	}
	return buildCmd
}

func createBuildCommand(imageBuilder, context, dockerFile, image string, imageBuildArgs ...string) (*exec.Cmd, error) {
	var args []string
	switch imageBuilder {
	case "docker", "podman":
		args = append(args, "build", "-f", dockerFile, "-t", image)
	case "buildah":
		args = append(args, "bud", "--format=docker", "-f", dockerFile, "-t", image)
	default:
		return nil, fmt.Errorf("%s is not supported image builder", imageBuilder)
	}

	for _, bargs := range imageBuildArgs {
		if bargs != "" {
			splitArgs, err := shlex.Split(bargs)
			if err != nil {
				return nil, fmt.Errorf("image-build-args is not parseable: %v", err)
			}
			args = append(args, splitArgs...)
		}
	}

	args = append(args, context)

	return exec.Command(imageBuilder, args...), nil
}

func buildFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("command %s requires exactly one argument", cmd.CommandPath())
	}

	image := args[0]
	projutil.MustInProjectRoot()

	if kbutil.HasProjectFile() {
		if err := doImageBuild("Dockerfile", image); err != nil {
			log.Fatalf("Failed to build image %s: %v", image, err)
		}
		return nil
	}

	// todo: remove when the legacy layout is no longer supported
	// note that the above if will no longer be required as well.
	if err := doLegacyBuild(image); err != nil {
		log.Fatalf("Failed to build image %s: %v", image, err)
	}
	return nil
}

// todo: remove when the legacy layout is no longer supported
// Deprecated: Used just for the legacy layout
// --
// doLegacyBuild will build projects with the legacy layout.
func doLegacyBuild(image string) error {
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

		if goBuildArgs != "" {
			splitArgs := strings.Fields(goBuildArgs)
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
	return doImageBuild("build/Dockerfile", image)
}

// doImageBuild will execute the build command for the given Dockerfile path and image
func doImageBuild(dockerFilePath, image string) error {
	log.Infof("Building OCI image %s", image)
	buildCmd, err := createBuildCommand(imageBuilder, ".", dockerFilePath, image, imageBuildArgs)
	if err != nil {
		return err
	}
	if err := projutil.ExecCmd(buildCmd); err != nil {
		return err
	}
	log.Info("Operator build complete.")
	return nil
}
