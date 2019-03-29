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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"
	"github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	namespacedManBuild string
	testLocationBuild  string
	enableTests        bool
	dockerBuildArgs    string
	genMultistage      bool
)

func NewCmd() *cobra.Command {
	buildCmd := &cobra.Command{
		Use:   "build <image>",
		Short: "Compiles code and builds artifacts",
		Long: `The operator-sdk build command compiles the code, builds the executables,
and generates Kubernetes manifests.

<image> is the container image to be built, e.g. "quay.io/example/operator:v0.0.1".
This image will be automatically set in the deployment manifests.

After build completes, the image would be built locally in docker. Then it needs to
be pushed to remote registry.
For example:
	$ operator-sdk build quay.io/example/operator:v0.0.1
	$ docker push quay.io/example/operator:v0.0.1
`,
		RunE: buildFunc,
	}
	buildCmd.Flags().BoolVar(&enableTests, "enable-tests", false, "Enable in-cluster testing by adding test binary to the image")
	buildCmd.Flags().BoolVar(&genMultistage, "gen-multistage", false, "Generate multistage build and test Dockerfiles")
	buildCmd.Flags().StringVar(&testLocationBuild, "test-location", "./test/e2e", "Location of tests")
	buildCmd.Flags().StringVar(&namespacedManBuild, "namespaced-manifest", "deploy/operator.yaml", "Path of namespaced resources manifest for tests")
	buildCmd.Flags().StringVar(&dockerBuildArgs, "docker-build-args", "", "Extra docker build arguments as one string such as \"--build-arg https_proxy=$https_proxy\"")
	return buildCmd
}

/*
 * verifyDeploymentImages checks image names of pod 0 in deployments found in the provided yaml file.
 * This is done because e2e tests require a namespaced manifest file to configure a namespace with
 * required resources. This function is intended to identify if a user used a different image name
 * for their operator in the provided yaml, which would result in the testing of the wrong operator
 * image. As it is possible for a namespaced yaml to have multiple deployments (such as the vault
 * operator, which depends on the etcd-operator), this is just a warning, not a fatal error.
 */
func verifyDeploymentImage(yamlFile []byte, imageName string) error {
	warningMessages := ""
	scanner := yamlutil.NewYAMLScanner(yamlFile)
	for scanner.Scan() {
		yamlSpec := scanner.Bytes()

		yamlMap := make(map[string]interface{})
		err := yaml.Unmarshal(yamlSpec, &yamlMap)
		if err != nil {
			return fmt.Errorf("could not unmarshal YAML namespaced spec: (%v)", err)
		}
		kind, ok := yamlMap["kind"].(string)
		if !ok {
			return fmt.Errorf("yaml manifest file contains a 'kind' field that is not a string")
		}
		if kind == "Deployment" {
			// this is ugly and hacky; we should probably make this cleaner
			nestedMap, ok := yamlMap["spec"].(map[string]interface{})
			if !ok {
				continue
			}
			nestedMap, ok = nestedMap["template"].(map[string]interface{})
			if !ok {
				continue
			}
			nestedMap, ok = nestedMap["spec"].(map[string]interface{})
			if !ok {
				continue
			}
			containersArray, ok := nestedMap["containers"].([]interface{})
			if !ok {
				continue
			}
			for _, item := range containersArray {
				image, ok := item.(map[string]interface{})["image"].(string)
				if !ok {
					continue
				}
				if image != imageName {
					warningMessages = fmt.Sprintf("%s\nWARNING: Namespace manifest contains a deployment with image %v, which does not match the name of the image being built: %v", warningMessages, image, imageName)
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to verify deployment image: (%v)", err)
	}
	if warningMessages == "" {
		return nil
	}
	return errors.New(warningMessages)
}

func verifyTestManifest(image string) error {
	namespacedBytes, err := ioutil.ReadFile(namespacedManBuild)
	if err != nil {
		return fmt.Errorf("could not read namespaced manifest: (%v)", err)
	}

	err = verifyDeploymentImage(namespacedBytes, image)
	// the error from verifyDeploymentImage is just a warning, not fatal error
	if err != nil {
		log.Warn(err)
	}
	return nil
}

func buildFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("command %s requires exactly one argument", cmd.CommandPath())
	}
	projutil.MustInProjectRoot()

	image := args[0]
	baseImageName := image
	if enableTests {
		baseImageName += "-intermediate"
	}
	var dbArgs []string
	if dockerBuildArgs != "" {
		splitArgs := regexp.MustCompile("--build-arg(=)?").ReplaceAllString(dockerBuildArgs, "")
		dbArgs = strings.Fields(splitArgs)
	}
	absProjectPath := projutil.MustGetwd()
	projectName := filepath.Base(absProjectPath)

	log.Infof("Building Docker image %s", baseImageName)

	// The runtime environment may have docker v17.05+, in which case the SDK
	// can build the binary within a container in a multistage pipeline.
	// Otherwise the binary will be built on the host and COPY'd into the
	// resulting image.
	buildDockerfile, err := makeDockerfileIfMultistage(filepath.Join(scaffold.BuildDir, scaffold.DockerfileFile))
	if err != nil {
		return err
	}
	if projutil.IsOperatorGo() && !projutil.IsDockerfileMultistage(buildDockerfile) {
		if err := buildOperatorBinary(); err != nil {
			return fmt.Errorf("failed to build operator binary: (%v)", err)
		}
	}
	err = projutil.DockerBuild(buildDockerfile, baseImageName, dbArgs...)
	if err != nil {
		if enableTests {
			return fmt.Errorf("failed to output intermediate image %s: (%v)", image, err)
		}
		return fmt.Errorf("failed to output build image %s: (%v)", image, err)
	}

	if enableTests {
		if !projutil.IsDockerMultistage() {
			return fmt.Errorf("in-cluster tests are only available on hosts with Docker v17.05+")
		}

		// If a user is using an older sdk repo as their library, make sure they
		// have the required build files.
		testDockerfile := filepath.Join(scaffold.BuildTestDir, scaffold.DockerfileFile)
		_, err = os.Stat(testDockerfile)
		if (err != nil && os.IsNotExist(err)) ||
			(projutil.IsOperatorGo() && !projutil.IsDockerfileMultistage(testDockerfile)) {

			log.Info("Generating build manifests for test-framework.")

			cfg := &input.Config{
				AbsProjectPath: absProjectPath,
				ProjectName:    projectName,
			}

			s := &scaffold.Scaffold{}
			t := projutil.GetOperatorType()
			switch t {
			case projutil.OperatorTypeGo:
				cfg.Repo = projutil.CheckAndGetProjectGoPkg()
				err = s.Execute(cfg,
					&scaffold.TestFrameworkDockerfile{
						Input: input.Input{IfExistsAction: input.Overwrite},
					},
					&scaffold.GoTestScript{},
					&scaffold.TestPod{Image: image, TestNamespaceEnv: test.TestNamespaceEnv},
				)
			case projutil.OperatorTypeAnsible:
				return fmt.Errorf("test scaffolding for Ansible Operators is not implemented")
			case projutil.OperatorTypeHelm:
				return fmt.Errorf("test scaffolding for Helm Operators is not implemented")
			default:
				return fmt.Errorf("unknown operator type '%v'", t)
			}

			if err != nil {
				return fmt.Errorf("test framework manifest scaffold failed: (%v)", err)
			}
		}

		log.Infof("Building test Docker image %s", image)

		// Tests require docker v17.05+ anyway so we don't need to conditionally
		// scaffold a multistage Dockerfile.
		err = projutil.DockerBuild(testDockerfile, image,
			append(dbArgs,
				"TESTDIR="+testLocationBuild,
				"BASEIMAGE="+baseImageName,
				"NAMESPACEDMAN="+namespacedManBuild)...)
		if err != nil {
			return fmt.Errorf("failed to output test image %s: (%v)", image, err)
		}
		// Check image name of deployments in namespaced manifest
		if err := verifyTestManifest(image); err != nil {
			return nil
		}
	}

	log.Info("Operator build complete.")
	return nil
}

// makeDockerfileIfMultistage is effectively a function for migrating
// single-stage to multistage Dockerfiles. makeDockerfileIfMultistage scaffolds
// a multistage Dockerfile for Go operators on hosts with docker v17.05+ if a
// multistage Dockerfile is not already present at path dockerfile. The newly
// scaffolded file is named 'multistage.Dockerfile', which users are expected
// to rename to 'Dockerfile'. Users will see a warning if docker v17.05+ is
// present but they haven't set the --gen-multistage flag in
// `operator-sdk build...`
func makeDockerfileIfMultistage(dockerfile string) (string, error) {
	if !projutil.IsOperatorGo() || !projutil.IsDockerMultistage() {
		return dockerfile, nil
	}

	if !projutil.IsDockerfileMultistage(dockerfile) {
		msDockerfile := "multistage." + scaffold.DockerfileFile
		if !genMultistage {
			log.Warnf(`Project uses a non-multistage Dockerfile but the present docker version
supports multistage builds. Run operator-sdk build with --gen-multistage to write
a multistage Dockerfile to '%s' and build it. Rename this
file to '%s' to avoid this warning.`,
				filepath.Join(scaffold.BuildDir, msDockerfile),
				dockerfile)

		} else {
			absProjectPath := projutil.MustGetwd()
			cfg := &input.Config{
				AbsProjectPath: absProjectPath,
				ProjectName:    filepath.Base(absProjectPath),
				Repo:           projutil.CheckAndGetProjectGoPkg(),
			}

			dockerfile = filepath.Join(scaffold.BuildDir, msDockerfile)
			d := &scaffold.Dockerfile{
				Multistage: true,
				Input: input.Input{
					Path:           dockerfile,
					IfExistsAction: input.Overwrite,
				},
			}
			err := (&scaffold.Scaffold{}).Execute(cfg, d)
			if err != nil {
				return "", fmt.Errorf("failed to write %s: (%v)", msDockerfile, err)
			}
		}
	}

	return dockerfile, nil
}

// buildOperatorBinary builds the operator binary locally.
func buildOperatorBinary() error {
	absProjectPath := projutil.MustGetwd()
	binName := filepath.Base(absProjectPath)
	goFlags := []string{"build",
		"-gcflags", "all=-trimpath=${GOPATH}",
		"-asmflags", "all=-trimpath=${GOPATH}",
		"-o", filepath.Join(absProjectPath, scaffold.BuildBinDir, binName),
		filepath.Join(projutil.CheckAndGetProjectGoPkg(), scaffold.ManagerDir),
	}

	cmd := exec.Command("go", goFlags...)
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	return projutil.ExecCmd(cmd)
}
