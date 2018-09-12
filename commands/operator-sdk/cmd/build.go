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
	globalManBuild     string
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
	buildCmd.Flags().BoolVarP(&enableTests, "enable-tests", "e", false, "Enable in-cluster testing by adding test binary to the image")
	buildCmd.Flags().StringVarP(&testLocationBuild, "test-location", "t", "./test/e2e", "Location of tests")
	buildCmd.Flags().StringVarP(&namespacedManBuild, "namespaced", "n", "", "Path of namespaced resources for tests")
	buildCmd.Flags().StringVarP(&globalManBuild, "global", "g", "deploy/crd.yaml", "Path of global resources for tests")
	return buildCmd
}

func parseRoles(yamlFile []byte) ([]byte, error) {
	res := make([]byte, 0)
	yamlSplit := bytes.Split(yamlFile, []byte("\n---\n"))
	for _, yamlSpec := range yamlSplit {
		yamlMap := make(map[string]interface{})
		err := yaml.Unmarshal(yamlSpec, &yamlMap)
		if err != nil {
			return nil, err
		}
		if yamlMap["kind"].(string) == "Role" {
			ruleBytes, err := yaml.Marshal(yamlMap["rules"])
			if err != nil {
				return nil, err
			}
			res = append(res, ruleBytes...)
		}
	}
	return res, nil
}

func verifyDeploymentImage(yamlFile []byte, imageName string) string {
	warningMessages := ""
	yamlSplit := bytes.Split(yamlFile, []byte("\n---\n"))
	for _, yamlSpec := range yamlSplit {
		yamlMap := make(map[string]interface{})
		err := yaml.Unmarshal(yamlSpec, &yamlMap)
		if err != nil {
			fmt.Printf("WARNING: Could not unmarshal yaml namespaced spec")
			return ""
		}
		if yamlMap["kind"].(string) == "Deployment" {
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
	return warningMessages
}

func renderTestManifests(image string) string {
	namespacedRolesBytes := make([]byte, 0)
	if namespacedManBuild == "" {
		os.Mkdir("deploy/test", os.FileMode(int(0775)))
		namespacedManBuild = "deploy/test/namespace-manifests.yaml"
		rbac, err := ioutil.ReadFile("deploy/rbac.yaml")
		if err != nil {
			log.Fatalf("could not find rbac manifest: %v", err)
		}
		operator, err := ioutil.ReadFile("deploy/operator.yaml")
		if err != nil {
			log.Fatalf("could not find operator manifest: %v", err)
		}
		combined := append(rbac, []byte("\n---\n")...)
		combined = append(combined, operator...)
		err = ioutil.WriteFile(namespacedManBuild, combined, os.FileMode(int(0664)))
		if err != nil {
			log.Fatalf("could not create temporary namespaced manifest file: %v", err)
		}
		defer func() {
			err := os.Remove(namespacedManBuild)
			if err != nil {
				log.Fatalf("could not delete temporary namespace manifest file")
			}
		}()
	}
	namespacedBytes, err := ioutil.ReadFile(namespacedManBuild)
	if err != nil {
		log.Fatalf("could not read rbac manifest: %v", err)
	}
	namespacedRolesBytes, err = parseRoles(namespacedBytes)
	if err != nil {
		log.Fatalf("could not parse namespaced manifest file for rbac roles: %v", err)
	}
	genWarning := verifyDeploymentImage(namespacedBytes, image)
	global, err := ioutil.ReadFile(globalManBuild)
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to read global manifest: (%v)", err))
	}
	c := cmdutil.GetConfig()
	if err = generator.RenderTestYaml(c, string(global), string(namespacedRolesBytes), image); err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to generate deploy/test-gen.yaml: (%v)", err))
	}
	return genWarning
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

	genWarning := ""
	image := args[0]
	intermediateImageName := image
	if enableTests {
		genWarning = renderTestManifests(image)
		intermediateImageName += "-intermediate"
	}
	dbcmd := exec.Command("docker", "build", ".", "-f", "tmp/build/Dockerfile", "-t", intermediateImageName)
	o, err = dbcmd.CombinedOutput()
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to output build image %v: (%v)", intermediateImageName, string(o)))
	}
	fmt.Fprintln(os.Stdout, string(o))

	if enableTests {
		testDbcmd := exec.Command("docker", "build", ".", "-f", "tmp/build/test-framework/Dockerfile", "-t", image, "--build-arg", "NAMESPACEDMAN="+namespacedManBuild, "--build-arg", "BASEIMAGE="+intermediateImageName)
		o, err = testDbcmd.CombinedOutput()
		if err != nil {
			cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to output build image %v: (%v)", image, string(o)))
		}
		fmt.Fprintln(os.Stdout, string(o))
		if genWarning != "" {
			fmt.Printf("%s\n", genWarning)
		}
	}
}
