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

	"github.com/operator-framework/operator-sdk/commands/operator-sdk/cmd/cmdutil"
	cmdError "github.com/operator-framework/operator-sdk/commands/operator-sdk/error"
	"github.com/operator-framework/operator-sdk/pkg/generator"

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

func renderTestManifest(image string) {
	namespacedBytes, err := ioutil.ReadFile(namespacedManBuild)
	if err != nil {
		log.Fatalf("could not read namespaced manifest: %v", err)
	}
	if err = generator.RenderTestYaml(cmdutil.GetConfig(), image); err != nil {
		log.Fatalf("failed to generate deploy/test-pod.yaml: (%v)", err)
	}
	err = verifyDeploymentImage(namespacedBytes, image)
	// the error from verifyDeploymentImage is just a warning, not fatal error
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}

const (
	build      = "./tmp/build/build.sh"
	configYaml = "./config/config.yaml"
)

func buildFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		cmdError.ExitWithError(cmdError.ExitBadArgs, fmt.Errorf("build command needs exactly 1 argument"))
	}

	bcmd := exec.Command(build)
	bcmd.Env = append(os.Environ(), fmt.Sprintf("TEST_LOCATION=%v", testLocationBuild))
	bcmd.Env = append(bcmd.Env, fmt.Sprintf("ENABLE_TESTS=%v", enableTests))
	o, err := bcmd.CombinedOutput()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to build: (%v)", string(o)))
	}
	fmt.Fprintln(os.Stdout, string(o))

	image := args[0]
	baseImageName := image
	if enableTests {
		baseImageName += "-intermediate"
	}
	dbcmd := exec.Command("docker", "build", ".", "-f", "tmp/build/Dockerfile", "-t", baseImageName)
	o, err = dbcmd.CombinedOutput()
	if err != nil {
		if enableTests {
			cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to build intermediate image for %s image: (%s)", image, string(o)))
		} else {
			cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to output build image %s: (%s)", image, string(o)))
		}
	}
	fmt.Fprintln(os.Stdout, string(o))

	if enableTests {
		testDbcmd := exec.Command("docker", "build", ".", "-f", "tmp/build/test-framework/Dockerfile", "-t", image, "--build-arg", "NAMESPACEDMAN="+namespacedManBuild, "--build-arg", "BASEIMAGE="+baseImageName)
		o, err = testDbcmd.CombinedOutput()
		if err != nil {
			cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to output build image %v: (%v)", image, string(o)))
		}
		fmt.Fprintln(os.Stdout, string(o))
		// create test-pod.yaml as well as check image name of deployments in namespaced manifest
		renderTestManifest(image)
	}
}
