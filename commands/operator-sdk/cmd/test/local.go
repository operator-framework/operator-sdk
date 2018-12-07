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

package cmdtest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
)

var deployTestDir = filepath.Join(scaffold.DeployDir, "test")

type testLocalConfig struct {
	kubeconfig        string
	globalManPath     string
	namespacedManPath string
	goTestFlags       string
	namespace         string
	upLocal           bool
	noSetup           bool
	image             string
}

var tlConfig testLocalConfig

func NewTestLocalCmd() *cobra.Command {
	testCmd := &cobra.Command{
		Use:   "local <path to tests directory> [flags]",
		Short: "Run End-To-End tests locally",
		Run:   testLocalFunc,
	}
	testCmd.Flags().StringVar(&tlConfig.kubeconfig, "kubeconfig", "", "Kubeconfig path")
	testCmd.Flags().StringVar(&tlConfig.globalManPath, "global-manifest", "", "Path to manifest for Global resources (e.g. CRD manifests)")
	testCmd.Flags().StringVar(&tlConfig.namespacedManPath, "namespaced-manifest", "", "Path to manifest for per-test, namespaced resources (e.g. RBAC and Operator manifest)")
	testCmd.Flags().StringVar(&tlConfig.goTestFlags, "go-test-flags", "", "Additional flags to pass to go test")
	testCmd.Flags().StringVar(&tlConfig.namespace, "namespace", "", "If non-empty, single namespace to run tests in")
	testCmd.Flags().BoolVar(&tlConfig.upLocal, "up-local", false, "Enable running operator locally with go run instead of as an image in the cluster")
	testCmd.Flags().BoolVar(&tlConfig.noSetup, "no-setup", false, "Disable test resource creation")
	testCmd.Flags().StringVar(&tlConfig.image, "image", "", "Use a different operator image from the one specified in the namespaced manifest")

	return testCmd
}

func testLocalFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		log.Fatal("operator-sdk test local requires exactly 1 argument")
	}
	if (tlConfig.noSetup && tlConfig.globalManPath != "") || (tlConfig.noSetup && tlConfig.namespacedManPath != "") {
		log.Fatal("the global-manifest and namespaced-manifest flags cannot be enabled at the same time as the no-setup flag")
	}

	if tlConfig.upLocal && tlConfig.namespace == "" {
		log.Fatal("must specify a namespace to run in when -up-local flag is set")
	}

	log.Info("Testing operator locally.")

	// if no namespaced manifest path is given, combine deploy/service_account.yaml, deploy/role.yaml, deploy/role_binding.yaml and deploy/operator.yaml
	if tlConfig.namespacedManPath == "" && !tlConfig.noSetup {
		err := os.MkdirAll(deployTestDir, os.FileMode(fileutil.DefaultDirFileMode))
		if err != nil {
			log.Fatalf("could not create %s: (%v)", deployTestDir, err)
		}
		tlConfig.namespacedManPath = filepath.Join(deployTestDir, "namespace-manifests.yaml")
		combined := []byte{}
		if !tlConfig.upLocal {
			sa, err := ioutil.ReadFile(filepath.Join(scaffold.DeployDir, scaffold.ServiceAccountYamlFile))
			if err != nil {
				log.Warnf("could not find the serviceaccount manifest: (%v)", err)
			}
			role, err := ioutil.ReadFile(filepath.Join(scaffold.DeployDir, scaffold.RoleYamlFile))
			if err != nil {
				log.Warnf("could not find role manifest: (%v)", err)
			}
			roleBinding, err := ioutil.ReadFile(filepath.Join(scaffold.DeployDir, scaffold.RoleBindingYamlFile))
			if err != nil {
				log.Warnf("could not find role_binding manifest: (%v)", err)
			}
			operator, err := ioutil.ReadFile(filepath.Join(scaffold.DeployDir, scaffold.OperatorYamlFile))
			if err != nil {
				log.Fatalf("could not find operator manifest: (%v)", err)
			}
			combined = combineManifests(combined, sa)
			combined = combineManifests(combined, role)
			combined = combineManifests(combined, roleBinding)
			combined = append(combined, operator...)
		}
		err = ioutil.WriteFile(tlConfig.namespacedManPath, combined, os.FileMode(fileutil.DefaultFileMode))
		if err != nil {
			log.Fatalf("could not create temporary namespaced manifest file: (%v)", err)
		}
		defer func() {
			err := os.Remove(tlConfig.namespacedManPath)
			if err != nil {
				log.Fatalf("could not delete temporary namespace manifest file: (%v)", err)
			}
		}()
	}
	if tlConfig.globalManPath == "" && !tlConfig.noSetup {
		err := os.MkdirAll(deployTestDir, os.FileMode(fileutil.DefaultDirFileMode))
		if err != nil {
			log.Fatalf("could not create %s: (%v)", deployTestDir, err)
		}
		tlConfig.globalManPath = filepath.Join(deployTestDir, "global-manifests.yaml")
		files, err := ioutil.ReadDir(scaffold.CrdsDir)
		if err != nil {
			log.Fatalf("could not read deploy directory: (%v)", err)
		}
		var combined []byte
		for _, file := range files {
			if strings.HasSuffix(file.Name(), "crd.yaml") {
				fileBytes, err := ioutil.ReadFile(filepath.Join(scaffold.CrdsDir, file.Name()))
				if err != nil {
					log.Fatalf("could not read file %s: (%v)", filepath.Join(scaffold.CrdsDir, file.Name()), err)
				}
				if combined == nil {
					combined = []byte{}
				} else {
					combined = append(combined, []byte("\n---\n")...)
				}
				combined = append(combined, fileBytes...)
			}
		}
		err = ioutil.WriteFile(tlConfig.globalManPath, combined, os.FileMode(fileutil.DefaultFileMode))
		if err != nil {
			log.Fatalf("could not create temporary global manifest file: (%v)", err)
		}
		defer func() {
			err := os.Remove(tlConfig.globalManPath)
			if err != nil {
				log.Fatalf("could not delete global manifest file: (%v)", err)
			}
		}()
	}
	if tlConfig.noSetup {
		err := os.MkdirAll(deployTestDir, os.FileMode(fileutil.DefaultDirFileMode))
		if err != nil {
			log.Fatalf("could not create %s: (%v)", deployTestDir, err)
		}
		tlConfig.namespacedManPath = filepath.Join(deployTestDir, "empty.yaml")
		tlConfig.globalManPath = filepath.Join(deployTestDir, "empty.yaml")
		emptyBytes := []byte{}
		err = ioutil.WriteFile(tlConfig.globalManPath, emptyBytes, os.FileMode(fileutil.DefaultFileMode))
		if err != nil {
			log.Fatalf("could not create empty manifest file: (%v)", err)
		}
		defer func() {
			err := os.Remove(tlConfig.globalManPath)
			if err != nil {
				log.Fatalf("could not delete empty manifest file: (%v)", err)
			}
		}()
	}
	if tlConfig.image != "" {
		if err := replaceImage(tlConfig.namespacedManPath, tlConfig.image); err != nil {
			log.Fatalf("failed to overwrite operator image in the namespaced manifest: %v", err)
		}
	}
	testArgs := []string{"test", args[0] + "/..."}
	if tlConfig.kubeconfig != "" {
		testArgs = append(testArgs, "-"+test.KubeConfigFlag, tlConfig.kubeconfig)
	}
	testArgs = append(testArgs, "-"+test.NamespacedManPathFlag, tlConfig.namespacedManPath)
	testArgs = append(testArgs, "-"+test.GlobalManPathFlag, tlConfig.globalManPath)
	testArgs = append(testArgs, "-"+test.ProjRootFlag, projutil.MustGetwd())
	// if we do the append using an empty go flags, it inserts an empty arg, which causes
	// any later flags to be ignored
	if tlConfig.goTestFlags != "" {
		testArgs = append(testArgs, strings.Split(tlConfig.goTestFlags, " ")...)
	}
	if tlConfig.namespace != "" || tlConfig.noSetup {
		testArgs = append(testArgs, "-"+test.SingleNamespaceFlag, "-parallel=1")
	}
	if tlConfig.upLocal {
		testArgs = append(testArgs, "-"+test.LocalOperatorFlag)
	}
	dc := exec.Command("go", testArgs...)
	dc.Env = append(os.Environ(), fmt.Sprintf("%v=%v", test.TestNamespaceEnv, tlConfig.namespace))
	dc.Dir = projutil.MustGetwd()
	dc.Stdout = os.Stdout
	dc.Stderr = os.Stderr
	err := dc.Run()
	if err != nil {
		log.Fatalf("failed to exec `go %s`: (%v)", strings.Join(testArgs, " "), err)
	}

	log.Info("Local operator test successfully completed.")
}

// combineManifests combines a given manifest with a base manifest and adds yaml
// style separation. Nothing is appended if the manifest is empty.
func combineManifests(base, manifest []byte) []byte {
	if len(manifest) > 0 {
		base = append(base, manifest...)
		return append(base, []byte("\n---\n")...)
	}
	return base
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
	yamlSplit := bytes.Split(yamlFile, []byte("\n---\n"))
	for _, yamlSpec := range yamlSplit {
		if string(yamlSpec) == "" {
			continue
		}
		decoded := make(map[string]interface{})
		err = yaml.Unmarshal(yamlSpec, &decoded)
		if err != nil {
			return err
		}
		kind, ok := decoded["kind"].(string)
		if !ok || kind != "Deployment" {
			newManifest = combineManifests(newManifest, yamlSpec)
			continue
		}
		if foundDeployment {
			return fmt.Errorf("cannot use `image` flag on namespaced manifest with more than 1 deployment")
		}
		foundDeployment = true
		scheme := runtime.NewScheme()
		// scheme for client go
		cgoscheme.AddToScheme(scheme)
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
		newManifest = combineManifests(newManifest, updatedYamlSpec)
	}
	return ioutil.WriteFile(manifestPath, newManifest, fileutil.DefaultFileMode)
}
