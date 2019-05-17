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

package scaffold

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/config"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type BuildCmd struct {
	Image          string
	ImageBuildArgs string
	ImageBuilder   string
}

func (c *BuildCmd) Run() error {

	goBuildEnv := append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
	// If CGO_ENABLED is not set, set it to '0'.
	if _, ok := os.LookupEnv("CGO_ENABLED"); !ok {
		goBuildEnv = append(goBuildEnv, "CGO_ENABLED=0")
	}

	goTrimFlags := []string{"-gcflags", "all=-trimpath=${GOPATH}", "-asmflags", "all=-trimpath=${GOPATH}"}
	absProjectPath := projutil.MustGetwd()
	projectName := filepath.Base(absProjectPath)

	// Don't need to build Go code if a non-Go Operator.
	if projutil.IsOperatorGo() {
		opts := projutil.GoCmdOptions{
			BinName:     filepath.Join(absProjectPath, scaffold.BuildBinDir, projectName),
			PackagePath: filepath.Join(viper.GetString(config.RepoOpt), scaffold.ManagerDir),
			Args:        goTrimFlags,
			Env:         goBuildEnv,
			GoMod:       projutil.IsDepManagerGoMod(),
		}
		if err := projutil.GoBuild(opts); err != nil {
			return fmt.Errorf("failed to build operator binary: (%v)", err)
		}
	}

	log.Infof("Building OCI image %s", c.Image)

	bc, err := createBuildCommand(c.ImageBuilder, ".", "build/Dockerfile", c.Image, c.ImageBuildArgs)
	if err != nil {
		return err
	}

	if err := projutil.ExecCmd(bc); err != nil {
		return fmt.Errorf("failed to output build image %s: (%v)", c.Image, err)
	}

	log.Info("Operator build complete.")
	return nil
}

func createBuildCommand(imageBuilder, context, dockerFile, image string, imageBuildArgs ...string) (*exec.Cmd, error) {
	var args []string
	switch imageBuilder {
	case "docker":
		args = append(args, "build", "-f", dockerFile, "-t", image)
	case "buildah":
		args = append(args, "bud", "--format=docker", "-f", dockerFile, "-t", image)
	default:
		return nil, fmt.Errorf("%s is not supported image builder", imageBuilder)
	}

	for _, bargs := range imageBuildArgs {
		if bargs != "" {
			splitArgs := strings.Fields(bargs)
			args = append(args, splitArgs...)
		}
	}

	args = append(args, context)

	return exec.Command(imageBuilder, args...), nil
}
