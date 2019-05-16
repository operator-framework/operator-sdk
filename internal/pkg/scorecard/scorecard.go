// Copyright 2019 The Operator-SDK Authors
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

package scorecard

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	k8sInternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"

	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olminstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ConfigOpt                 = "config"
	NamespaceOpt              = "namespace"
	KubeconfigOpt             = "kubeconfig"
	InitTimeoutOpt            = "init-timeout"
	OlmDeployedOpt            = "olm-deployed"
	CSVPathOpt                = "csv-path"
	BasicTestsOpt             = "basic-tests"
	OLMTestsOpt               = "olm-tests"
	NamespacedManifestOpt     = "namespaced-manifest"
	GlobalManifestOpt         = "global-manifest"
	CRManifestOpt             = "cr-manifest"
	ProxyImageOpt             = "proxy-image"
	ProxyPullPolicyOpt        = "proxy-pull-policy"
	CRDsDirOpt                = "crds-dir"
	OutputFormatOpt           = "output"
	PluginDirOpt              = "plugin-dir"
	JSONOutputFormat          = "json"
	HumanReadableOutputFormat = "human-readable"
)

const (
	basicOperator  = "Basic Operator"
	olmIntegration = "OLM Integration"
)

var (
	kubeconfig     *rest.Config
	dynamicDecoder runtime.Decoder
	runtimeClient  client.Client
	restMapper     *restmapper.DeferredDiscoveryRESTMapper
	deploymentName string
	proxyPodGlobal *v1.Pod
	cleanupFns     []cleanupFn
)

const (
	scorecardPodName       = "operator-scorecard-test"
	scorecardContainerName = "scorecard-proxy"
)

// make a global logger for scorecard
var (
	logReadWriter io.ReadWriter
	log           = logrus.New()
)

func runTests() ([]scapiv1alpha1.ScorecardOutput, error) {
	defer func() {
		if err := cleanupScorecard(); err != nil {
			log.Errorf("Failed to cleanup resources: (%v)", err)
		}
	}()

	var (
		tmpNamespaceVar string
		err             error
	)
	kubeconfig, tmpNamespaceVar, err = k8sInternal.GetKubeconfigAndNamespace(viper.GetString(KubeconfigOpt))
	if err != nil {
		return nil, fmt.Errorf("failed to build the kubeconfig: %v", err)
	}
	if viper.GetString(NamespaceOpt) == "" {
		viper.Set(NamespaceOpt, tmpNamespaceVar)
	}
	scheme := runtime.NewScheme()
	// scheme for client go
	if err := cgoscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add client-go scheme to client: (%v)", err)
	}
	// api extensions scheme (CRDs)
	if err := extscheme.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add failed to add extensions api scheme to client: (%v)", err)
	}
	// olm api (CS
	if err := olmapiv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add failed to add oml api scheme (CSVs) to client: (%v)", err)
	}
	dynamicDecoder = serializer.NewCodecFactory(scheme).UniversalDeserializer()
	// if a user creates a new CRD, we need to be able to reset the rest mapper
	// temporary kubeclient to get a cached discovery
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to get a kubeclient: %v", err)
	}
	cachedDiscoveryClient := cached.NewMemCacheClient(kubeclient.Discovery())
	restMapper = restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()
	runtimeClient, _ = client.New(kubeconfig, client.Options{Scheme: scheme, Mapper: restMapper})

	csv := &olmapiv1alpha1.ClusterServiceVersion{}
	if viper.GetBool(OLMTestsOpt) {
		yamlSpec, err := ioutil.ReadFile(viper.GetString(CSVPathOpt))
		if err != nil {
			return nil, fmt.Errorf("failed to read csv: %v", err)
		}
		if err = yaml.Unmarshal(yamlSpec, csv); err != nil {
			return nil, fmt.Errorf("error getting ClusterServiceVersion: %v", err)
		}
	}

	// Extract operator manifests from the CSV if olm-deployed is set.
	if viper.GetBool(OlmDeployedOpt) {
		// Get deploymentName from the deployment manifest within the CSV.
		strat, err := (&olminstall.StrategyResolver{}).UnmarshalStrategy(csv.Spec.InstallStrategy)
		if err != nil {
			return nil, err
		}
		stratDep, ok := strat.(*olminstall.StrategyDetailsDeployment)
		if !ok {
			return nil, fmt.Errorf("expected StrategyDetailsDeployment, got strategy of type %T", strat)
		}
		deploymentName = stratDep.DeploymentSpecs[0].Name
		// Get the proxy pod, which should have been created with the CSV.
		proxyPodGlobal, err = getPodFromDeployment(deploymentName, viper.GetString(NamespaceOpt))
		if err != nil {
			return nil, err
		}

		// Create a temporary CR manifest from metadata if one is not provided.
		crJSONStr, ok := csv.ObjectMeta.Annotations["alm-examples"]
		if ok && viper.GetString(CRManifestOpt) == "" {
			var crs []interface{}
			if err = json.Unmarshal([]byte(crJSONStr), &crs); err != nil {
				return nil, err
			}
			// TODO: run scorecard against all CR's in CSV.
			cr := crs[0]
			crJSONBytes, err := json.Marshal(cr)
			if err != nil {
				return nil, err
			}
			crYAMLBytes, err := yaml.JSONToYAML(crJSONBytes)
			if err != nil {
				return nil, err
			}
			crFile, err := ioutil.TempFile("", "cr.yaml")
			if err != nil {
				return nil, err
			}
			if _, err := crFile.Write(crYAMLBytes); err != nil {
				return nil, err
			}
			viper.Set(CRManifestOpt, crFile.Name())
			defer func() {
				err := os.Remove(viper.GetString(CRManifestOpt))
				if err != nil {
					log.Errorf("Could not delete temporary CR manifest file: (%v)", err)
				}
			}()
		}

	} else {
		// If no namespaced manifest path is given, combine
		// deploy/{service_account,role.yaml,role_binding,operator}.yaml.
		if viper.GetString(NamespacedManifestOpt) == "" {
			file, err := yamlutil.GenerateCombinedNamespacedManifest(scaffold.DeployDir)
			if err != nil {
				return nil, err
			}
			viper.Set(NamespacedManifestOpt, file.Name())
			defer func() {
				err := os.Remove(viper.GetString(NamespacedManifestOpt))
				if err != nil {
					log.Errorf("Could not delete temporary namespace manifest file: (%v)", err)
				}
			}()
		}
		// If no global manifest is given, combine all CRD's in the given CRD's dir.
		if viper.GetString(GlobalManifestOpt) == "" {
			gMan, err := yamlutil.GenerateCombinedGlobalManifest(viper.GetString(CRDsDirOpt))
			if err != nil {
				return nil, err
			}
			viper.Set(GlobalManifestOpt, gMan.Name())
			defer func() {
				err := os.Remove(viper.GetString(GlobalManifestOpt))
				if err != nil {
					log.Errorf("Could not delete global manifest file: (%v)", err)
				}
			}()
		}
	}

	crs := viper.GetStringSlice(CRManifestOpt)
	// check if there are duplicate CRs
	gvks := []schema.GroupVersionKind{}
	for _, cr := range crs {
		file, err := ioutil.ReadFile(cr)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %s", cr)
		}
		newGVKs, err := getGVKs(file)
		if err != nil {
			return nil, fmt.Errorf("could not get GVKs for resource(s) in file: %s, due to error: (%v)", cr, err)
		}
		gvks = append(gvks, newGVKs...)
	}
	dupMap := make(map[schema.GroupVersionKind]bool)
	for _, gvk := range gvks {
		if _, ok := dupMap[gvk]; ok {
			log.Warnf("Duplicate gvks in CR list detected (%s); results may be inaccurate", gvk)
		}
		dupMap[gvk] = true
	}

	var pluginResults []scapiv1alpha1.ScorecardOutput
	var suites []TestSuite
	for _, cr := range crs {
		// TODO: Change built-in tests into plugins
		// Run built-in tests.
		fmt.Printf("Running for cr: %s\n", cr)
		if !viper.GetBool(OlmDeployedOpt) {
			if err := createFromYAMLFile(viper.GetString(GlobalManifestOpt)); err != nil {
				return nil, fmt.Errorf("failed to create global resources: %v", err)
			}
			if err := createFromYAMLFile(viper.GetString(NamespacedManifestOpt)); err != nil {
				return nil, fmt.Errorf("failed to create namespaced resources: %v", err)
			}
		}
		if err := createFromYAMLFile(cr); err != nil {
			return nil, fmt.Errorf("failed to create cr resource: %v", err)
		}
		obj, err := yamlToUnstructured(cr)
		if err != nil {
			return nil, fmt.Errorf("failed to decode custom resource manifest into object: %s", err)
		}
		if err := waitUntilCRStatusExists(obj); err != nil {
			return nil, fmt.Errorf("failed waiting to check if CR status exists: %v", err)
		}
		if viper.GetBool(BasicTestsOpt) {
			conf := BasicTestConfig{
				Client:   runtimeClient,
				CR:       obj,
				ProxyPod: proxyPodGlobal,
			}
			basicTests := NewBasicTestSuite(conf)
			basicTests.Run(context.TODO())
			suites = append(suites, *basicTests)
		}
		if viper.GetBool(OLMTestsOpt) {
			conf := OLMTestConfig{
				Client:   runtimeClient,
				CR:       obj,
				CSV:      csv,
				CRDsDir:  viper.GetString(CRDsDirOpt),
				ProxyPod: proxyPodGlobal,
			}
			olmTests := NewOLMTestSuite(conf)
			olmTests.Run(context.TODO())
			suites = append(suites, *olmTests)
		}
		// set up clean environment for every CR
		if err := cleanupScorecard(); err != nil {
			log.Errorf("Failed to cleanup resources: (%v)", err)
		}
		// reset cleanup functions
		cleanupFns = []cleanupFn{}
		// clear name of operator deployment
		deploymentName = ""
	}
	suites, err = MergeSuites(suites)
	if err != nil {
		return nil, fmt.Errorf("failed to merge test suite results: %v", err)
	}
	for _, suite := range suites {
		// convert to ScorecardOutput format
		// will add log when basic and olm tests are separated into plugins
		pluginResults = append(pluginResults, TestSuitesToScorecardOutput([]TestSuite{suite}, ""))
	}
	// Run plugins
	pluginDir := viper.GetString(PluginDirOpt)
	if dir, err := os.Stat(pluginDir); err != nil || !dir.IsDir() {
		log.Warnf("Plugin directory not found; skipping plugin tests: %v", err)
		return pluginResults, nil
	}
	if err := os.Chdir(pluginDir); err != nil {
		return nil, fmt.Errorf("failed to chdir into scorecard plugin directory: %v", err)
	}
	// executable files must be in "bin" subdirectory
	files, err := ioutil.ReadDir("bin")
	if err != nil {
		return nil, fmt.Errorf("failed to list files in %s/bin: %v", pluginDir, err)
	}
	for _, file := range files {
		cmd := exec.Command("./bin/" + file.Name())
		stdout := &bytes.Buffer{}
		cmd.Stdout = stdout
		stderr := &bytes.Buffer{}
		cmd.Stderr = stderr
		err := cmd.Run()
		if err != nil {
			name := fmt.Sprintf("Failed Plugin: %s", file.Name())
			description := fmt.Sprintf("Plugin with file name `%s` failed", file.Name())
			logs := fmt.Sprintf("%s:\nStdout: %s\nStderr: %s", err, string(stdout.Bytes()), string(stderr.Bytes()))
			pluginResults = append(pluginResults, failedPlugin(name, description, logs))
			// output error to main logger as well for human-readable output
			log.Errorf("Plugin `%s` failed with error (%v)", file.Name(), err)
			continue
		}
		// parse output and add to suites
		result := scapiv1alpha1.ScorecardOutput{}
		err = json.Unmarshal(stdout.Bytes(), &result)
		if err != nil {
			name := fmt.Sprintf("Plugin output invalid: %s", file.Name())
			description := fmt.Sprintf("Plugin with file name %s did not produce valid ScorecardOutput JSON", file.Name())
			logs := fmt.Sprintf("Stdout: %s\nStderr: %s", string(stdout.Bytes()), string(stderr.Bytes()))
			pluginResults = append(pluginResults, failedPlugin(name, description, logs))
			log.Errorf("Output from plugin `%s` failed to unmarshal with error (%v)", file.Name(), err)
			continue
		}
		stderrString := string(stderr.Bytes())
		if len(stderrString) != 0 {
			log.Warn(stderrString)
		}
		pluginResults = append(pluginResults, result)
	}
	return pluginResults, nil
}

func ScorecardTests(cmd *cobra.Command, args []string) error {
	if err := initConfig(); err != nil {
		return err
	}
	if err := validateScorecardFlags(); err != nil {
		return err
	}
	cmd.SilenceUsage = true
	pluginOutputs, err := runTests()
	if err != nil {
		return err
	}
	totalScore := 0.0
	// Update the state for the tests
	for _, suite := range pluginOutputs {
		for idx, res := range suite.Results {
			suite.Results[idx] = UpdateSuiteStates(res)
		}
	}
	if viper.GetString(OutputFormatOpt) == HumanReadableOutputFormat {
		numSuites := 0
		for _, plugin := range pluginOutputs {
			for _, suite := range plugin.Results {
				fmt.Printf("%s:\n", suite.Name)
				for _, result := range suite.Tests {
					fmt.Printf("\t%s: %d/%d\n", result.Name, result.EarnedPoints, result.MaximumPoints)
				}
				totalScore += float64(suite.TotalScore)
				numSuites++
			}
		}
		totalScore = totalScore / float64(numSuites)
		fmt.Printf("\nTotal Score: %.0f%%\n", totalScore)
		// TODO: We can probably use some helper functions to clean up these quadruple nested loops
		// Print suggestions
		for _, plugin := range pluginOutputs {
			for _, suite := range plugin.Results {
				for _, result := range suite.Tests {
					for _, suggestion := range result.Suggestions {
						// 33 is yellow (specifically, the same shade of yellow that logrus uses for warnings)
						fmt.Printf("\x1b[%dmSUGGESTION:\x1b[0m %s\n", 33, suggestion)
					}
				}
			}
		}
		// Print errors
		for _, plugin := range pluginOutputs {
			for _, suite := range plugin.Results {
				for _, result := range suite.Tests {
					for _, err := range result.Errors {
						// 31 is red (specifically, the same shade of red that logrus uses for errors)
						fmt.Printf("\x1b[%dmERROR:\x1b[0m %s\n", 31, err)
					}
				}
			}
		}
	}
	if viper.GetString(OutputFormatOpt) == JSONOutputFormat {
		log, err := ioutil.ReadAll(logReadWriter)
		if err != nil {
			return fmt.Errorf("failed to read log buffer: %v", err)
		}
		scTest := CombineScorecardOutput(pluginOutputs, string(log))
		// Pretty print so users can also read the json output
		bytes, err := json.MarshalIndent(scTest, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", string(bytes))
	}
	return nil
}

func initConfig() error {
	// viper/cobra already has flags parsed at this point; we can check if a config file flag is set
	if viper.GetString(ConfigOpt) != "" {
		// Use config file from the flag.
		viper.SetConfigFile(viper.GetString(ConfigOpt))
	} else {
		viper.AddConfigPath(projutil.MustGetwd())
		// using SetConfigName allows users to use a .yaml, .json, or .toml file
		viper.SetConfigName(".osdk-scorecard")
	}

	if err := viper.ReadInConfig(); err == nil {
		// configure logger output before logging anything
		err := configureLogger()
		if err != nil {
			return err
		}
		log.Info("Using config file: ", viper.ConfigFileUsed())
	} else {
		err := configureLogger()
		if err != nil {
			return err
		}
		log.Warn("Could not load config file; using flags")
	}
	return nil
}

func configureLogger() error {
	if viper.GetString(OutputFormatOpt) == HumanReadableOutputFormat {
		logReadWriter = os.Stdout
	} else if viper.GetString(OutputFormatOpt) == JSONOutputFormat {
		logReadWriter = &bytes.Buffer{}
	} else {
		return fmt.Errorf("invalid output format: %s", viper.GetString(OutputFormatOpt))
	}
	log.SetOutput(logReadWriter)
	return nil
}

func validateScorecardFlags() error {
	if !viper.GetBool(OlmDeployedOpt) && viper.GetStringSlice(CRManifestOpt) == nil {
		return errors.New("cr-manifest config option must be set")
	}
	if !viper.GetBool(BasicTestsOpt) && !viper.GetBool(OLMTestsOpt) {
		return errors.New("at least one test type must be set")
	}
	if viper.GetBool(OLMTestsOpt) && viper.GetString(CSVPathOpt) == "" {
		return fmt.Errorf("csv-path must be set if olm-tests is enabled")
	}
	if viper.GetBool(OlmDeployedOpt) && viper.GetString(CSVPathOpt) == "" {
		return fmt.Errorf("csv-path must be set if olm-deployed is enabled")
	}
	pullPolicy := viper.GetString(ProxyPullPolicyOpt)
	if pullPolicy != "Always" && pullPolicy != "Never" && pullPolicy != "PullIfNotPresent" {
		return fmt.Errorf("invalid proxy pull policy: (%s); valid values: Always, Never, PullIfNotPresent", pullPolicy)
	}
	// this is already being checked in configure logger; may be unnecessary
	outputFormat := viper.GetString(OutputFormatOpt)
	if outputFormat != HumanReadableOutputFormat && outputFormat != JSONOutputFormat {
		return fmt.Errorf("invalid output format (%s); valid values: %s, %s", outputFormat, HumanReadableOutputFormat, JSONOutputFormat)
	}
	return nil
}

func getGVKs(yamlFile []byte) ([]schema.GroupVersionKind, error) {
	var gvks []schema.GroupVersionKind

	scanner := yamlutil.NewYAMLScanner(yamlFile)
	for scanner.Scan() {
		yamlSpec := scanner.Bytes()

		obj := &unstructured.Unstructured{}
		jsonSpec, err := yaml.YAMLToJSON(yamlSpec)
		if err != nil {
			return nil, fmt.Errorf("could not convert yaml file to json: %v", err)
		}
		if err := obj.UnmarshalJSON(jsonSpec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal object spec: (%v)", err)
		}
		gvks = append(gvks, obj.GroupVersionKind())
	}
	return gvks, nil
}

func failedPlugin(name, desc, log string) scapiv1alpha1.ScorecardOutput {
	return scapiv1alpha1.ScorecardOutput{
		Results: []scapiv1alpha1.ScorecardSuiteResult{{
			Name:        name,
			Description: desc,
			Error:       1,
			Log:         log,
		},
		},
	}
}
