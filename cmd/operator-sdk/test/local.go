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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	internalk8sutil "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/test"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/yaml"
)

var deployTestDir = filepath.Join(scaffold.DeployDir, "test")

type testLocalConfig struct {
	kubeconfig        string
	globalManPath     string
	namespacedManPath string
	goTestFlags       string
	moleculeTestFlags string
	// TODO: remove before 1.0.0
	// Namespace is deprecated
	namespace          string
	operatorNamespace  string
	watchNamespace     string
	image              string
	localOperatorFlags string
	upLocal            bool
	noSetup            bool
	debug              bool
	skipCleanupOnError bool
}

var tlConfig testLocalConfig

func newTestLocalCmd() *cobra.Command {
	testCmd := &cobra.Command{
		Use:   "local <path to tests directory> [flags]",
		Short: "Run End-To-End tests locally",
		RunE:  testLocalFunc,
	}
	testCmd.Flags().StringVar(&tlConfig.kubeconfig, "kubeconfig", "", "Kubeconfig path")
	testCmd.Flags().StringVar(&tlConfig.globalManPath, "global-manifest", "",
		"Path to manifest for Global resources (e.g. CRD manifests)")
	testCmd.Flags().StringVar(&tlConfig.namespacedManPath, "namespaced-manifest", "",
		"Path to manifest for per-test, namespaced resources (e.g. RBAC and Operator manifest)")
	testCmd.Flags().StringVar(&tlConfig.goTestFlags, "go-test-flags", "",
		"Additional flags to pass to go test")
	testCmd.Flags().StringVar(&tlConfig.moleculeTestFlags, "molecule-test-flags", "",
		"Additional flags to pass to molecule test")
	// TODO: remove before 1.0.0. Namespace is deprecated
	testCmd.Flags().StringVar(&tlConfig.namespace, "namespace", "",
		"(Deprecated: use --operator-namespace instead) If non-empty, single namespace to run tests in")
	testCmd.Flags().StringVar(&tlConfig.operatorNamespace, "operator-namespace", "",
		"Namespace where the operator will be deployed, CRs will be created and tests will be executed "+
			"(By default it will be in the default namespace defined in the kubeconfig)")
	testCmd.Flags().StringVar(&tlConfig.watchNamespace, "watch-namespace", "",
		"(only valid with --up-local) Namespace where the operator watches for changes."+
			" Set \"\" for AllNamespaces, set \"ns1,ns2\" for MultiNamespace"+
			"(if not set then watches Operator Namespace")
	testCmd.Flags().BoolVar(&tlConfig.upLocal, "up-local", false,
		"Enable running operator locally with go run instead of as an image in the cluster")
	testCmd.Flags().BoolVar(&tlConfig.noSetup, "no-setup", false, "Disable test resource creation")
	testCmd.Flags().BoolVar(&tlConfig.debug, "debug", false, "Enable debug-level logging")
	testCmd.Flags().StringVar(&tlConfig.image, "image", "",
		"Use a different operator image from the one specified in the namespaced manifest")
	testCmd.Flags().StringVar(&tlConfig.localOperatorFlags, "local-operator-flags", "",
		"The flags that the operator needs (while using --up-local). Example: \"--flag1 value1 --flag2=value2\"")
	testCmd.Flags().BoolVar(&tlConfig.skipCleanupOnError, "skip-cleanup-error", false,
		"If set as true, the cleanup function responsible to remove all artifacts "+
			"will be skipped if an error is faced.")

	return testCmd
}

func testLocalFunc(cmd *cobra.Command, args []string) error {
	//TODO: remove before 1.0.0
	// set --operator-namespace flag if the --namespace flag is set
	// (only if --operator-namespace flag is not set)
	if cmd.Flags().Changed("namespace") {
		log.Info("--namespace is deprecated; use --operator-namespace instead.")
		if !cmd.Flags().Changed("operator-namespace") {
			err := cmd.Flags().Set("operator-namespace", tlConfig.namespace)
			return err
		}
	}
	switch t := projutil.GetOperatorType(); t {
	case projutil.OperatorTypeGo:
		return testLocalGoFunc(cmd, args)
	case projutil.OperatorTypeAnsible:
		return testLocalAnsibleFunc()
	case projutil.OperatorTypeHelm:
		return fmt.Errorf("`test local` for Helm operators is not implemented")
	}
	return projutil.ErrUnknownOperatorType{}
}

func testLocalAnsibleFunc() error {
	projutil.MustInProjectRoot()
	testArgs := []string{}
	if tlConfig.debug {
		testArgs = append(testArgs, "--debug")
	}
	testArgs = append(testArgs, "test", "-s", "test-local")

	if tlConfig.moleculeTestFlags != "" {
		testArgs = append(testArgs, strings.Split(tlConfig.moleculeTestFlags, " ")...)
	}

	dc := exec.Command("molecule", testArgs...)
	dc.Env = append(os.Environ(), fmt.Sprintf("%v=%v", test.TestOperatorNamespaceEnv, tlConfig.operatorNamespace))
	dc.Dir = projutil.MustGetwd()
	if err := projutil.ExecCmd(dc); err != nil {
		log.Fatal(err)
	}
	return nil
}

func testLocalGoFunc(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("command %s requires exactly one argument", cmd.CommandPath())
	}
	if (tlConfig.noSetup && tlConfig.globalManPath != "") ||
		(tlConfig.noSetup && tlConfig.namespacedManPath != "") {
		return fmt.Errorf("the global-manifest and namespaced-manifest flags cannot be enabled" +
			" at the same time as the no-setup flag")
	}

	if tlConfig.upLocal && tlConfig.operatorNamespace == "" {
		return fmt.Errorf("must specify a namespace with operator-namespace flag to run in when --up-local flag is set")
	}
	if !tlConfig.upLocal && cmd.Flags().Changed("watch-namespace") {
		return fmt.Errorf("--watch-namespace not valid without -up-local flag")
	}

	log.Info("Testing operator locally.")

	// if no namespaced manifest path is given, combine deploy/service_account.yaml, deploy/role.yaml,
	// deploy/role_binding.yaml and deploy/operator.yaml
	if tlConfig.namespacedManPath == "" && !tlConfig.noSetup {
		if !tlConfig.upLocal {
			file, err := internalk8sutil.GenerateCombinedNamespacedManifest(scaffold.DeployDir)
			if err != nil {
				return err
			}
			tlConfig.namespacedManPath = file.Name()
		} else {
			file, err := ioutil.TempFile("", "empty.yaml")
			if err != nil {
				return fmt.Errorf("could not create empty manifest file: %v", err)
			}
			tlConfig.namespacedManPath = file.Name()
			emptyBytes := []byte{}
			if err := file.Chmod(os.FileMode(fileutil.DefaultFileMode)); err != nil {
				return fmt.Errorf("could not chown temporary namespaced manifest file: %v", err)
			}
			if _, err := file.Write(emptyBytes); err != nil {
				return fmt.Errorf("could not write temporary namespaced manifest file: %v", err)
			}
			if err := file.Close(); err != nil {
				return err
			}
		}
		defer func() {
			err := os.Remove(tlConfig.namespacedManPath)
			if err != nil {
				log.Errorf("Could not delete temporary namespace manifest file: (%v)", err)
			}
		}()
	}
	if tlConfig.globalManPath == "" && !tlConfig.noSetup {
		file, err := internalk8sutil.GenerateCombinedGlobalManifest(scaffold.CRDsDir)
		if err != nil {
			return err
		}
		tlConfig.globalManPath = file.Name()
		defer func() {
			err := os.Remove(tlConfig.globalManPath)
			if err != nil {
				log.Errorf("Could not delete global manifest file: (%v)", err)
			}
		}()
	}
	if tlConfig.noSetup {
		err := os.MkdirAll(deployTestDir, os.FileMode(fileutil.DefaultDirFileMode))
		if err != nil {
			return fmt.Errorf("could not create %s: %v", deployTestDir, err)
		}
		tlConfig.namespacedManPath = filepath.Join(deployTestDir, "empty.yaml")
		tlConfig.globalManPath = filepath.Join(deployTestDir, "empty.yaml")
		emptyBytes := []byte{}
		err = ioutil.WriteFile(tlConfig.globalManPath, emptyBytes, os.FileMode(fileutil.DefaultFileMode))
		if err != nil {
			return fmt.Errorf("could not create empty manifest file: %v", err)
		}
		defer func() {
			err := os.Remove(tlConfig.globalManPath)
			if err != nil {
				log.Errorf("Could not delete empty manifest file: (%v)", err)
			}
		}()
	}
	if tlConfig.image != "" {
		err := replaceImage(tlConfig.namespacedManPath, tlConfig.image)
		if err != nil {
			return fmt.Errorf("failed to overwrite operator image in the namespaced manifest: %v", err)
		}
	}
	testArgs := []string{
		"-" + test.NamespacedManPathFlag, tlConfig.namespacedManPath,
		"-" + test.GlobalManPathFlag, tlConfig.globalManPath,
		"-" + test.ProjRootFlag, projutil.MustGetwd(),
	}
	if tlConfig.kubeconfig != "" {
		testArgs = append(testArgs, "-"+test.KubeConfigFlag, tlConfig.kubeconfig)
	}
	// if we do the append using an empty go flags, it inserts an empty arg, which causes
	// any later flags to be ignored
	if tlConfig.goTestFlags != "" {
		testArgs = append(testArgs, strings.Split(tlConfig.goTestFlags, " ")...)
	}
	if tlConfig.operatorNamespace != "" || tlConfig.noSetup {
		testArgs = append(testArgs, "-parallel=1")
	}
	env := os.Environ()
	if tlConfig.operatorNamespace != "" {
		env = append(
			env,
			fmt.Sprintf("%v=%v", test.TestOperatorNamespaceEnv, tlConfig.operatorNamespace),
		)
	}

	if cmd.Flags().Changed("watch-namespace") {
		env = append(
			env,
			fmt.Sprintf("%v=%v", test.TestWatchNamespaceEnv, tlConfig.watchNamespace),
		)
	}

	if tlConfig.upLocal {
		env = append(env, fmt.Sprintf("%s=%s", k8sutil.ForceRunModeEnv, k8sutil.LocalRunMode))
		testArgs = append(testArgs, "-"+test.LocalOperatorFlag)
		if tlConfig.localOperatorFlags != "" {
			testArgs = append(testArgs, "-"+test.LocalOperatorArgs, tlConfig.localOperatorFlags)
		}
	}
	testArgs = append(testArgs, fmt.Sprintf("-%s=%t", test.SkipCleanupOnErrorFlag, tlConfig.skipCleanupOnError))
	opts := projutil.GoTestOptions{
		GoCmdOptions: projutil.GoCmdOptions{
			PackagePath: args[0] + "/...",
			Env:         env,
			Dir:         projutil.MustGetwd(),
		},
		TestBinaryArgs: testArgs,
	}
	if err := projutil.GoTest(opts); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		log.Fatalf("Failed to build test binary: %v", err)
	}
	log.Info("Local operator test successfully completed.")
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
	scanner := internalk8sutil.NewYAMLScanner(bytes.NewBuffer(yamlFile))
	for scanner.Scan() {
		yamlSpec := scanner.Bytes()

		decoded := make(map[string]interface{})
		err = yaml.Unmarshal(yamlSpec, &decoded)
		if err != nil {
			return err
		}
		kind, ok := decoded["kind"].(string)
		if !ok || kind != "Deployment" {
			newManifest = internalk8sutil.CombineManifests(newManifest, yamlSpec)
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
		var dep appsv1.Deployment
		switch o := obj.(type) {
		case *appsv1.Deployment:
			dep = *o
		default:
			return fmt.Errorf("error in replaceImage switch case; could not convert runtime.Object" +
				" to deployment")
		}
		if len(dep.Spec.Template.Spec.Containers) != 1 {
			return fmt.Errorf("cannot use `image` flag on namespaced manifest containing more" +
				" than 1 container in the operator deployment")
		}
		dep.Spec.Template.Spec.Containers[0].Image = image
		updatedYamlSpec, err := yaml.Marshal(dep)
		if err != nil {
			return fmt.Errorf("failed to convert deployment object back to yaml: %v", err)
		}
		newManifest = internalk8sutil.CombineManifests(newManifest, updatedYamlSpec)
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to scan %s: %v", manifestPath, err)
	}

	return ioutil.WriteFile(manifestPath, newManifest, fileutil.DefaultFileMode)
}
