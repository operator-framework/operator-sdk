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

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

var (
	namespacedManBuild string
	testLocationBuild  string
	enableTests        bool
)

func NewBuildCmd() *cobra.Command {
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
		Run: buildFunc,
	}
	buildCmd.Flags().BoolVar(&enableTests, "enable-tests", false, "Enable in-cluster testing by adding test binary to the image")
	buildCmd.Flags().StringVar(&testLocationBuild, "test-location", "./test/e2e", "Location of tests")
	buildCmd.Flags().StringVar(&namespacedManBuild, "namespaced-manifest", "deploy/operator.yaml", "Path of namespaced resources manifest for tests")
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
	yamlSplit := bytes.Split(yamlFile, []byte("\n---\n"))
	for _, yamlSpec := range yamlSplit {
		yamlMap := make(map[string]interface{})
		err := yaml.Unmarshal(yamlSpec, &yamlMap)
		if err != nil {
			log.Fatal("Could not unmarshal yaml namespaced spec")
		}
		kind, ok := yamlMap["kind"].(string)
		if !ok {
			log.Fatal("Yaml manifest file contains a 'kind' field that is not a string")
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
	if warningMessages == "" {
		return nil
	}
	return errors.New(warningMessages)
}

func verifyTestManifest(image string) {
	namespacedBytes, err := ioutil.ReadFile(namespacedManBuild)
	if err != nil {
		log.Fatalf("could not read namespaced manifest: %v", err)
	}

	err = verifyDeploymentImage(namespacedBytes, image)
	// the error from verifyDeploymentImage is just a warning, not fatal error
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}

func buildFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		log.Fatalf("build command needs exactly 1 argument")
	}
	projutil.MustInProjectRoot()

	image := args[0]
	baseImageName := image
	if enableTests {
		baseImageName += "-intermediate"
	}

	buildCmd := exec.Command("docker", "build", ".",
		"-f", filepath.Join(scaffold.BuildDir, scaffold.DockerfileFile),
		"-t", baseImageName)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	err := buildCmd.Run()
	if err != nil {
		if enableTests {
			log.Fatalf("failed to build intermediate image for %s image: %v", image, err)
		} else {
			log.Fatalf("failed to output build image %s: %v", image, err)
		}
	}

	if enableTests {
		// if a user is using an older sdk repo as their library, make sure they have required build files
		testDockerfile := filepath.Join(scaffold.BuildTestDir, scaffold.DockerfileFile)
		_, err = os.Stat(testDockerfile)
		if err != nil && os.IsNotExist(err) {

			absProjectPath := projutil.MustGetwd()
			cfg := &input.Config{
				Repo:           projutil.CheckAndGetProjectGoPkg(),
				AbsProjectPath: absProjectPath,
				ProjectName:    filepath.Base(absProjectPath),
			}

			s := &scaffold.Scaffold{}
			err = s.Execute(cfg,
				&scaffold.TestFrameworkDockerfile{},
				&scaffold.GoTestScript{},
				&scaffold.TestPod{Image: image, TestNamespaceEnv: test.TestNamespaceEnv},
			)
			if err != nil {
				log.Fatalf("build scaffold failed: (%v)", err)
			}
		}

		testBuildCmd := exec.Command("docker", "build", ".",
			"-f", testDockerfile,
			"-t", image,
			"--build-arg", "TESTDIR="+testLocationBuild,
			"--build-arg", "BASEIMAGE="+baseImageName,
			"--build-arg", "NAMESPACEDMAN="+namespacedManBuild)
		testBuildCmd.Stdout = os.Stdout
		testBuildCmd.Stderr = os.Stderr
		err = testBuildCmd.Run()
		if err != nil {
			log.Fatalf("failed to output build image %s: %v", image, err)
		}
		// Check image name of deployments in namespaced manifest
		verifyTestManifest(image)
	}
}
