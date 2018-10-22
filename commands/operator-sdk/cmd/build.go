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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/coreos/go-semver/semver"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"
	catalog "github.com/operator-framework/operator-sdk/pkg/scaffold/olm-catalog"
	"github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	namespacedManBuild string
	testLocationBuild  string
	enableTests        bool
	genCSV             bool
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

Supplying an argument to '--gen-csv' directs the SDK to create a
ClusterServiceVersion manifest.
`,
		Run: buildFunc,
	}
	buildCmd.Flags().BoolVar(&genCSV, "gen-csv", false, "Directs the SDK to compose a CSV.\t\nConfigure this process by writing a config file 'deploy/olm-catalog/csv-config.yaml'")
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
			log.Fatalf("could not unmarshal yaml namespaced spec: (%v)", err)
		}
		kind, ok := yamlMap["kind"].(string)
		if !ok {
			log.Fatal("yaml manifest file contains a 'kind' field that is not a string")
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
		log.Fatalf("could not read namespaced manifest: (%v)", err)
	}

	err = verifyDeploymentImage(namespacedBytes, image)
	// the error from verifyDeploymentImage is just a warning, not fatal error
	if err != nil {
		log.Warn(err)
	}
}

func checkImageFormatAndGetVersion(image string) string {
	splitImage := strings.Split(image, ":")
	if len(splitImage) == 1 {
		log.Fatal("no operator version supplied in image name")
	}
	opVer := splitImage[1]
	if len(opVer) > 0 && opVer[0] == 'v' {
		opVer = opVer[1:len(opVer)]
	}
	if _, err := semver.NewVersion(opVer); err != nil {
		// TODO: is this functionality ok, or should we allow users to version their operators with 'latest'?
		log.Fatalf("operator version '%s' is not a semantic version", opVer)
	}
	return opVer
}

func buildFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		log.Fatalf("build command needs exactly 1 argument")
	}
	image := args[0]
	opVer := checkImageFormatAndGetVersion(image)

	projutil.MustInProjectRoot()
	goBuildEnv := append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	wd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not identify current working directory: (%v)", err)
	}

	// Don't need to build go code if Ansible Operator
	if mainExists() {
		managerDir := filepath.Join(projutil.CheckAndGetProjectGoPkg(), scaffold.ManagerDir)
		outputBinName := filepath.Join(wd, scaffold.BuildBinDir, filepath.Base(wd))
		buildCmd := exec.Command("go", "build", "-o", outputBinName, managerDir)
		buildCmd.Env = goBuildEnv
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		err = buildCmd.Run()
		if err != nil {
			log.Fatalf("failed to build operator binary: (%v)", err)
		}
	}

	absProjectPath := projutil.MustGetwd()
	cfg := &input.Config{
		AbsProjectPath: absProjectPath,
		ProjectName:    filepath.Base(wd),
	}

	// Create a CSV if the user calls build with `--gen-csv`.
	if genCSV {
		s := &scaffold.Scaffold{}
		err = s.Execute(cfg,
			&catalog.Csv{OperatorVersion: opVer},
		)
		if err != nil {
			log.Fatalf("build catalog scaffold failed: (%v)", err)
		}
	}

	baseImageName := image
	if enableTests {
		baseImageName += "-intermediate"
	}

	log.Infof("Building Docker image %s", baseImageName)

	dbcmd := exec.Command("docker", "build", ".", "-f", "build/Dockerfile", "-t", baseImageName)
	dbcmd.Stdout = os.Stdout
	dbcmd.Stderr = os.Stderr
	err = dbcmd.Run()
	if err != nil {
		if enableTests {
			log.Fatalf("failed to output intermediate image %s: (%v)", image, err)
		} else {
			log.Fatalf("failed to output build image %s: (%v)", image, err)
		}
	}

	if enableTests {
		cfg.Repo = projutil.CheckAndGetProjectGoPkg()

		testBinary := filepath.Join(wd, scaffold.BuildBinDir, filepath.Base(wd)+"-test")
		buildTestCmd := exec.Command("go", "test", "-c", "-o", testBinary, testLocationBuild+"/...")
		buildTestCmd.Env = goBuildEnv
		buildTestCmd.Stdout = os.Stdout
		buildTestCmd.Stderr = os.Stderr
		err = buildTestCmd.Run()
		if err != nil {
			log.Fatalf("failed to build test binary: (%v)", err)
		}

		// if a user is using an older sdk repo as their library, make sure they have required build files
		testDockerfile := filepath.Join(scaffold.BuildTestDir, scaffold.DockerfileFile)
		_, err = os.Stat(testDockerfile)
		if err != nil && os.IsNotExist(err) {

			log.Info("Generating build manifests for test-framework.")

			s := &scaffold.Scaffold{}
			err = s.Execute(cfg,
				&scaffold.TestFrameworkDockerfile{},
				&scaffold.GoTestScript{},
				&scaffold.TestPod{Image: image, TestNamespaceEnv: test.TestNamespaceEnv},
			)
			if err != nil {
				log.Fatalf("test-framework manifest scaffold failed: (%v)", err)
			}
		}

		log.Infof("Building test Docker image %s", image)

		testDbcmd := exec.Command("docker", "build", ".", "-f", testDockerfile, "-t", image, "--build-arg", "NAMESPACEDMAN="+namespacedManBuild, "--build-arg", "BASEIMAGE="+baseImageName)
		testDbcmd.Stdout = os.Stdout
		testDbcmd.Stderr = os.Stderr
		err = testDbcmd.Run()
		if err != nil {
			log.Fatalf("failed to output test image %s: (%v)", image, err)
		}
		// Check image name of deployments in namespaced manifest
		verifyTestManifest(image)
	}

	log.Info("Operator build complete.")
}

func mainExists() bool {
	_, err := os.Stat(filepath.Join(scaffold.ManagerDir, scaffold.CmdFile))
	return err == nil
}
