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

package test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"
	"github.com/operator-framework/operator-sdk/pkg/config"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
)

type TestCmd struct {
	Namespace      string
	KubeconfigPath string
}

type LocalAnsibleCmd struct {
	TestCmd

	MoleculeTestFlags string
}

func (c *LocalAnsibleCmd) Run() error {
	projutil.MustInProjectRoot()
	testArgs := []string{}
	if viper.GetBool(config.VerboseOpt) {
		testArgs = append(testArgs, "--debug")
	}
	testArgs = append(testArgs, "test", "-s", "test-local")

	if c.MoleculeTestFlags != "" {
		testArgs = append(testArgs, strings.Split(c.MoleculeTestFlags, " ")...)
	}

	dc := exec.Command("molecule", testArgs...)
	dc.Env = append(os.Environ(), fmt.Sprintf("%v=%v", TestNamespaceEnv, c.Namespace))
	dc.Dir = projutil.MustGetwd()
	return projutil.ExecCmd(dc)
}

type LocalGoCmd struct {
	TestCmd

	TestPath          string
	GlobalManPath     string
	NamespacedManPath string
	GoTestFlags       string
	UpLocal           bool
	NoSetup           bool
	Image             string

	deployTestDir string
}

func (c *LocalGoCmd) Run() error {

	if err := c.validateFlags(); err != nil {
		return err
	}

	c.deployTestDir = filepath.Join(viper.GetString(config.DeployDirOpt), "test")

	log.Info("Testing operator locally.")

	// if no namespaced manifest path is given, combine deploy/service_account.yaml, deploy/role.yaml, deploy/role_binding.yaml and deploy/operator.yaml
	if c.NamespacedManPath == "" && !c.NoSetup {
		if !c.UpLocal {
			file, err := yamlutil.GenerateCombinedNamespacedManifest(viper.GetString(config.DeployDirOpt))
			if err != nil {
				return err
			}
			c.NamespacedManPath = file.Name()
		} else {
			file, err := ioutil.TempFile("", "empty.yaml")
			if err != nil {
				return fmt.Errorf("could not create empty manifest file: (%v)", err)
			}
			c.NamespacedManPath = file.Name()
			emptyBytes := []byte{}
			if err := file.Chmod(os.FileMode(fileutil.DefaultFileMode)); err != nil {
				return fmt.Errorf("could not chown temporary namespaced manifest file: (%v)", err)
			}
			if _, err := file.Write(emptyBytes); err != nil {
				return fmt.Errorf("could not write temporary namespaced manifest file: (%v)", err)
			}
			if err := file.Close(); err != nil {
				return err
			}
		}
		defer func() {
			err := os.Remove(c.NamespacedManPath)
			if err != nil {
				log.Errorf("Could not delete temporary namespace manifest file: (%v)", err)
			}
		}()
	}
	if c.GlobalManPath == "" && !c.NoSetup {
		file, err := yamlutil.GenerateCombinedGlobalManifest(viper.GetString(config.CRDsDirOpt))
		if err != nil {
			return err
		}
		c.GlobalManPath = file.Name()
		defer func() {
			err := os.Remove(c.GlobalManPath)
			if err != nil {
				log.Errorf("Could not delete global manifest file: (%v)", err)
			}
		}()
	}
	if c.NoSetup {
		err := os.MkdirAll(c.deployTestDir, os.FileMode(fileutil.DefaultDirFileMode))
		if err != nil {
			return fmt.Errorf("could not create %s: (%v)", c.deployTestDir, err)
		}
		c.NamespacedManPath = filepath.Join(c.deployTestDir, "empty.yaml")
		c.GlobalManPath = filepath.Join(c.deployTestDir, "empty.yaml")
		emptyBytes := []byte{}
		err = ioutil.WriteFile(c.GlobalManPath, emptyBytes, os.FileMode(fileutil.DefaultFileMode))
		if err != nil {
			return fmt.Errorf("could not create empty manifest file: (%v)", err)
		}
		defer func() {
			err := os.Remove(c.GlobalManPath)
			if err != nil {
				log.Errorf("Could not delete empty manifest file: (%v)", err)
			}
		}()
	}
	if c.Image != "" {
		err := replaceImage(c.NamespacedManPath, c.Image)
		if err != nil {
			return fmt.Errorf("failed to overwrite operator image in the namespaced manifest: %v", err)
		}
	}
	testArgs := []string{
		"-" + NamespacedManPathFlag, c.NamespacedManPath,
		"-" + GlobalManPathFlag, c.GlobalManPath,
		"-" + ProjRootFlag, projutil.MustGetwd(),
	}
	if c.KubeconfigPath != "" {
		testArgs = append(testArgs, "-"+KubeConfigFlag, c.KubeconfigPath)
	}
	// if we do the append using an empty go flags, it inserts an empty arg, which causes
	// any later flags to be ignored
	if c.GoTestFlags != "" {
		testArgs = append(testArgs, strings.Split(c.GoTestFlags, " ")...)
	}
	if c.Namespace != "" || c.NoSetup {
		testArgs = append(testArgs, "-"+SingleNamespaceFlag, "-parallel=1")
	}
	if c.UpLocal {
		testArgs = append(testArgs, "-"+LocalOperatorFlag)
	}
	opts := projutil.GoTestOptions{
		GoCmdOptions: projutil.GoCmdOptions{
			PackagePath: c.TestPath + "/...",
			Env:         append(os.Environ(), fmt.Sprintf("%v=%v", TestNamespaceEnv, c.Namespace)),
			Dir:         projutil.MustGetwd(),
			GoMod:       projutil.IsDepManagerGoMod(),
		},
		TestBinaryArgs: testArgs,
	}
	if err := projutil.GoTest(opts); err != nil {
		return fmt.Errorf("failed to build test binary: (%v)", err)
	}
	log.Info("Local operator test successfully completed.")
	return nil
}

func (c *LocalGoCmd) validateFlags() error {
	if (c.NoSetup && c.GlobalManPath != "") ||
		(c.NoSetup && c.NamespacedManPath != "") {
		return fmt.Errorf("the global-manifest and namespaced-manifest flags cannot be enabled at the same time as the no-setup flag")
	}

	if c.UpLocal && c.Namespace == "" {
		return fmt.Errorf("must specify a namespace to run in when -up-local flag is set")
	}
	return nil
}

// TODO: add support for multiple deployments and containers (user would have to
// provide extra information in that case)

// replaceImage searches for a deployment and replaces the image in the container
// to the one specified in the function call. The function will fail if the
// number of deployments is not equal to one or if the deployment has multiple
// containers
func replaceImage(manifestPath, image string) error {
	yamlFile, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	foundDeployment := false
	newManifest := []byte{}
	scanner := yamlutil.NewYAMLScanner(yamlFile)
	for scanner.Scan() {
		yamlSpec := scanner.Bytes()

		decoded := make(map[string]interface{})
		err = yaml.Unmarshal(yamlSpec, &decoded)
		if err != nil {
			return err
		}
		kind, ok := decoded["kind"].(string)
		if !ok || kind != "Deployment" {
			newManifest = yamlutil.CombineManifests(newManifest, yamlSpec)
			continue
		}
		if foundDeployment {
			return fmt.Errorf("cannot use `image` flag on namespaced manifest with more than 1 deployment")
		}
		foundDeployment = true
		scheme := runtime.NewScheme()
		// scheme for client go
		if err := cgoscheme.AddToScheme(scheme); err != nil {
			log.Fatalf("Failed to add client-go scheme to runtime client: (%v)", err)
		}
		dynamicDecoder := serializer.NewCodecFactory(scheme).UniversalDeserializer()

		obj, _, err := dynamicDecoder.Decode(yamlSpec, nil, nil)
		if err != nil {
			return err
		}
		dep := &appsv1.Deployment{}
		switch o := obj.(type) {
		case *appsv1.Deployment:
			dep = o
		default:
			return fmt.Errorf("error in replaceImage switch case; could not convert runtime.Object to deployment")
		}
		if len(dep.Spec.Template.Spec.Containers) != 1 {
			return fmt.Errorf("cannot use `image` flag on namespaced manifest containing more than 1 container in the operator deployment")
		}
		dep.Spec.Template.Spec.Containers[0].Image = image
		updatedYamlSpec, err := yaml.Marshal(dep)
		if err != nil {
			return fmt.Errorf("failed to convert deployment object back to yaml: %v", err)
		}
		newManifest = yamlutil.CombineManifests(newManifest, updatedYamlSpec)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan %s: (%v)", manifestPath, err)
	}

	return ioutil.WriteFile(manifestPath, newManifest, fileutil.DefaultFileMode)
}
