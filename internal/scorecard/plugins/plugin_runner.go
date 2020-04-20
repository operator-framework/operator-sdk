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

package scplugins

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/operator-framework/api/pkg/validation"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	schelpers "github.com/operator-framework/operator-sdk/internal/scorecard/helpers"
	internalk8sutil "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"

	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	cached "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type PluginType int

const (
	BasicOperator  PluginType = 0
	OLMIntegration PluginType = 1
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
	scorecardContainerName = "scorecard-proxy"
)

var log *logrus.Logger

func RunInternalPlugin(pluginType PluginType, config BasicAndOLMPluginConfig,
	logFile io.Writer) (scapiv1alpha2.ScorecardOutput, error) {

	// use stderr for logging not related to a single suite
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{DisableColors: true})
	log.SetOutput(logFile)

	if err := validateScorecardPluginFlags(config, pluginType); err != nil {
		return scapiv1alpha2.ScorecardOutput{}, err
	}
	defer func() {
		if err := cleanupScorecard(); err != nil {
			log.SetOutput(logFile)
			log.Errorf("Failed to cleanup resources: (%v)", err)
		}
	}()

	var tmpNamespaceVar string
	var err error
	kubeconfig, tmpNamespaceVar, err = internalk8sutil.GetKubeconfigAndNamespace(config.Kubeconfig)
	if err != nil {
		return scapiv1alpha2.ScorecardOutput{}, fmt.Errorf("failed to build the kubeconfig: %v", err)
	}

	if config.Namespace == "" {
		config.Namespace = tmpNamespaceVar
	}

	if err := setupRuntimeClient(); err != nil {
		return scapiv1alpha2.ScorecardOutput{}, err
	}

	csv := &olmapiv1alpha1.ClusterServiceVersion{}
	if pluginType == OLMIntegration || config.OLMDeployed {
		err := getCSV(config.CSVManifest, csv)
		if err != nil {
			return scapiv1alpha2.ScorecardOutput{}, err
		}
	}

	// Extract operator manifests from the CSV if olm-deployed is set.
	if config.OLMDeployed {
		// Get deploymentName from the deployment manifest within the CSV.
		deploymentName, err = getDeploymentName(csv.Spec.InstallStrategy)
		if err != nil {
			return scapiv1alpha2.ScorecardOutput{}, err
		}
		// Get the proxy pod, which should have been created with the CSV.
		proxyPodGlobal, err = getPodFromDeployment(deploymentName, config.Namespace)
		if err != nil {
			return scapiv1alpha2.ScorecardOutput{}, err
		}

		config.CRManifest, err = getCRFromCSV(config.CRManifest, csv.ObjectMeta.Annotations["alm-examples"],
			csv.GetName())
		if err != nil {
			return scapiv1alpha2.ScorecardOutput{}, err
		}

	} else {
		// If no namespaced manifest path is given, combine
		// deploy/{service_account,role.yaml,role_binding,operator}.yaml.
		if config.NamespacedManifest == "" {
			file, err := internalk8sutil.GenerateCombinedNamespacedManifest(scaffold.DeployDir)
			if err != nil {
				return scapiv1alpha2.ScorecardOutput{}, err
			}
			config.NamespacedManifest = file.Name()
			defer func() {
				err := os.Remove(config.NamespacedManifest)
				if err != nil {
					log.Errorf("Could not delete temporary namespace manifest file: (%v)", err)
				}
				config.NamespacedManifest = ""
			}()
		}
		// If no global manifest is given, combine all CRD's in the given CRD's dir.
		if config.GlobalManifest == "" {
			if config.CRDsDir == "" {
				config.CRDsDir = filepath.Join(scaffold.DeployDir, "crds")
			}
			gMan, err := internalk8sutil.GenerateCombinedGlobalManifest(config.CRDsDir)
			if err != nil {
				return scapiv1alpha2.ScorecardOutput{}, err
			}
			config.GlobalManifest = gMan.Name()
			defer func() {
				err := os.Remove(config.GlobalManifest)
				if err != nil {
					log.Errorf("Could not delete global manifest file: (%v)", err)
				}
				config.GlobalManifest = ""
			}()
		}
	}

	err = duplicateCRCheck(config.CRManifest)
	if err != nil {
		return scapiv1alpha2.ScorecardOutput{}, err
	}

	testResults := make([]schelpers.TestResult, 0)
	for _, cr := range config.CRManifest {
		crTestResults, _, err := runTests(csv, pluginType, config, cr, logFile)
		if err != nil {
			return scapiv1alpha2.ScorecardOutput{}, err
		}
		testResults = append(testResults, crTestResults...)
	}

	output := scapiv1alpha2.NewScorecardOutput()
	output.Log = ""

	for _, tr := range testResults {
		output.Results = append(output.Results, testResultToScorecardTestResult(tr))
	}

	return *output, nil
}

func ListInternalPlugin(pluginType PluginType, config BasicAndOLMPluginConfig) (scapiv1alpha2.ScorecardOutput, error) {
	testResults := make([]schelpers.TestResult, 0)
	tests := make([]schelpers.Test, 0)

	switch pluginType {
	case BasicOperator:
		conf := BasicTestConfig{}
		tests = append(tests, NewCheckSpecTest(conf))
		tests = append(tests, NewCheckStatusTest(conf))
		tests = append(tests, NewWritingIntoCRsHasEffectTest(conf))
	case OLMIntegration:
		conf := OLMTestConfig{}
		tests = append(tests, NewBundleValidationTest(conf))
		tests = append(tests, NewCRDsHaveValidationTest(conf))
		tests = append(tests, NewCRDsHaveResourcesTest(conf))
		tests = append(tests, NewSpecDescriptorsTest(conf))
		tests = append(tests, NewStatusDescriptorsTest(conf))
	}

	tests = applySelector(tests, config.Selector)

	for i := 0; i < len(tests); i++ {
		result := schelpers.TestResult{}
		result.State = scapiv1alpha2.PassState
		result.Test = tests[i]
		result.Suggestions = make([]string, 0)
		result.Errors = make([]error, 0)
		testResults = append(testResults, result)
	}

	output := scapiv1alpha2.NewScorecardOutput()
	output.Log = ""

	for _, tr := range testResults {
		output.Results = append(output.Results, testResultToScorecardTestResult(tr))
	}

	return *output, nil
}

func getStructShortName(obj interface{}) string {
	t := reflect.TypeOf(obj)
	return strings.ToLower(t.Name())
}

func setupRuntimeClient() error {
	scheme := runtime.NewScheme()
	// scheme for client go
	if err := cgoscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add client-go scheme to client: (%v)", err)
	}
	// api extensions scheme (CRDs)
	if err := extscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add failed to add extensions api scheme to client: (%v)", err)
	}
	// olm api (CSVs)
	if err := olmapiv1alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add failed to add oml api scheme (CSVs) to client: (%v)", err)
	}
	dynamicDecoder = serializer.NewCodecFactory(scheme).UniversalDeserializer()
	// if a user creates a new CRD, we need to be able to reset the rest mapper
	// temporary kubeclient to get a cached discovery
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get a kubeclient: %v", err)
	}
	cachedDiscoveryClient := cached.NewMemCacheClient(kubeclient.Discovery())
	restMapper = restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()
	runtimeClient, _ = client.New(kubeconfig, client.Options{Scheme: scheme, Mapper: restMapper})
	return nil
}

func getCSV(csvManifest string, csv *olmapiv1alpha1.ClusterServiceVersion) error {
	yamlSpec, err := ioutil.ReadFile(csvManifest)
	if err != nil {
		return fmt.Errorf("failed to read csv: %v", err)
	}
	if err = yaml.Unmarshal(yamlSpec, csv); err != nil {
		return fmt.Errorf("error getting ClusterServiceVersion: %v", err)
	}

	csvValidator := validation.ClusterServiceVersionValidator
	results := csvValidator.Validate(csv)
	for _, r := range results {
		if len(r.Errors) > 0 {
			var errorMsgs strings.Builder
			for _, e := range r.Errors {
				errorMsgs.WriteString(fmt.Sprintf("%s\n", e.Error()))
			}
			return fmt.Errorf("error validating ClusterServiceVersion: %s", errorMsgs.String())
		}
		for _, w := range r.Warnings {
			log.Warnf("CSV validation warning: type [%s] %s", w.Type, w.Detail)
		}
	}

	return nil
}

func getDeploymentName(strategy olmapiv1alpha1.NamedInstallStrategy) (string, error) {
	if len(strategy.StrategySpec.DeploymentSpecs) == 0 {
		return "", errors.New("no deployment specs in CSV")
	}
	return strategy.StrategySpec.DeploymentSpecs[0].Name, nil
}

func getCRFromCSV(currentCRMans []string, crJSONStr string, csvName string) ([]string, error) {
	finalCR := make([]string, 0)
	logCRMsg := false
	if crMans := currentCRMans; len(crMans) == 0 {
		// Create a temporary CR manifest from metadata if one is not provided.
		if crJSONStr != "" {
			var crs []interface{}
			if err := json.Unmarshal([]byte(crJSONStr), &crs); err != nil {
				return finalCR, fmt.Errorf("metadata.annotations['alm-examples'] in CSV %s"+
					"incorrectly formatted: %v", csvName, err)
			}
			if len(crs) == 0 {
				return finalCR, fmt.Errorf("no CRs found in metadata.annotations['alm-examples']"+
					" in CSV %s and cr-manifest config option not set", csvName)
			}
			// TODO: run scorecard against all CR's in CSV.
			cr := crs[0]
			logCRMsg = len(crs) > 1
			crJSONBytes, err := json.Marshal(cr)
			if err != nil {
				return finalCR, err
			}
			crYAMLBytes, err := yaml.JSONToYAML(crJSONBytes)
			if err != nil {
				return finalCR, err
			}
			crFile, err := ioutil.TempFile("", "*.cr.yaml")
			if err != nil {
				return finalCR, err
			}
			if _, err := crFile.Write(crYAMLBytes); err != nil {
				return finalCR, err
			}
			finalCR = []string{crFile.Name()}
			defer func() {
				for _, f := range finalCR {
					if err := os.Remove(f); err != nil {
						log.Errorf("Could not delete temporary CR manifest file: (%v)", err)
					}
				}
			}()
		} else {
			return finalCR, errors.New(
				"cr-manifest config option must be set if CSV has no metadata.annotations['alm-examples']")
		}
	} else {
		// TODO: run scorecard against all CR's in CSV.
		finalCR = []string{crMans[0]}
		logCRMsg = len(crMans) > 1
	}
	// Let users know that only the first CR is being tested.
	if logCRMsg {
		log.Infof("The scorecard does not support testing multiple CR's at once when run with --olm-deployed."+
			" Testing the first CR %s", finalCR[0])
	}
	return finalCR, nil
}

// Check if there are duplicate CRs
func duplicateCRCheck(crs []string) error {
	gvks := []schema.GroupVersionKind{}
	for _, cr := range crs {
		file, err := ioutil.ReadFile(cr)
		if err != nil {
			return fmt.Errorf("failed to read file: %s", cr)
		}
		newGVKs, err := getGVKs(file)
		if err != nil {
			return fmt.Errorf("could not get GVKs for resource(s) in file: %s, due to error: (%v)", cr, err)
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
	return nil
}

func runTests(csv *olmapiv1alpha1.ClusterServiceVersion, pluginType PluginType, cfg BasicAndOLMPluginConfig,
	cr string, logFile io.Writer) ([]schelpers.TestResult, string, error) {
	testResults := make([]schelpers.TestResult, 0)

	logReadWriter := &bytes.Buffer{}
	log.SetOutput(logReadWriter)
	log.Printf("Running for cr: %s", cr)

	if !cfg.OLMDeployed {
		if err := createFromYAMLFile(cfg, cfg.GlobalManifest); err != nil {
			return testResults, "", fmt.Errorf("failed to create global resources: %v", err)
		}
		if err := createFromYAMLFile(cfg, cfg.NamespacedManifest); err != nil {
			return testResults, "", fmt.Errorf("failed to create namespaced resources: %v", err)
		}
	}

	if err := createFromYAMLFile(cfg, cr); err != nil {
		return testResults, "", fmt.Errorf("failed to create cr resource: %v", err)
	}

	obj, err := yamlToUnstructured(cfg.Namespace, cr)
	if err != nil {
		return testResults, "", fmt.Errorf("failed to decode custom resource manifest into object: %s", err)
	}

	if err := waitUntilCRStatusExists(time.Second*time.Duration(cfg.InitTimeout), obj); err != nil {
		return testResults, "", fmt.Errorf("failed waiting to check if CR status exists: %v", err)
	}

	tests := make([]schelpers.Test, 0)

	switch pluginType {
	case BasicOperator:
		conf := BasicTestConfig{
			Client:   runtimeClient,
			CR:       obj,
			ProxyPod: proxyPodGlobal,
		}

		tests = append(tests, NewCheckSpecTest(conf))
		tests = append(tests, NewCheckStatusTest(conf))
		tests = append(tests, NewWritingIntoCRsHasEffectTest(conf))

	case OLMIntegration:
		conf := OLMTestConfig{
			Client:   runtimeClient,
			CR:       obj,
			CSV:      csv,
			CRDsDir:  cfg.CRDsDir,
			ProxyPod: proxyPodGlobal,
			Bundle:   cfg.Bundle,
		}

		tests = append(tests, NewBundleValidationTest(conf))
		tests = append(tests, NewCRDsHaveValidationTest(conf))
		tests = append(tests, NewCRDsHaveResourcesTest(conf))
		tests = append(tests, NewSpecDescriptorsTest(conf))
		tests = append(tests, NewStatusDescriptorsTest(conf))

	}

	tests = applySelector(tests, cfg.Selector)

	for _, test := range tests {
		testResults = append(testResults, *test.Run(context.TODO()))
	}

	var testResultsLog string
	logs, err := ioutil.ReadAll(logReadWriter)
	if err != nil {
		testResultsLog = fmt.Sprintf("failed to read log buffer: %v", err)
	} else {
		testResultsLog = string(logs)
	}

	// change logging back to main log
	log.SetOutput(logFile)
	// set up clean environment for every CR
	if err := cleanupScorecard(); err != nil {
		log.Errorf("Failed to cleanup resources: (%v)", err)
	}
	// reset cleanup functions
	cleanupFns = []cleanupFn{}
	// clear name of operator deployment
	deploymentName = ""

	return testResults, testResultsLog, nil
}

// applySelector apply label selectors removing tests that do not match
func applySelector(tests []schelpers.Test, selector labels.Selector) []schelpers.Test {
	for i := 0; i < len(tests); i++ {
		t := tests[i]
		if !selector.Matches(labels.Set(t.GetLabels())) {
			// Remove the test
			tests = append(tests[:i], tests[i+1:]...)
			i--
		}
	}
	return tests
}

// testResultToScorecardTestResult is a helper function for converting from the TestResult type
// to the ScorecardTestResult type
func testResultToScorecardTestResult(tr schelpers.TestResult) scapiv1alpha2.ScorecardTestResult {
	sctr := scapiv1alpha2.ScorecardTestResult{}
	sctr.State = tr.State
	sctr.Name = tr.Test.GetName()
	sctr.Description = tr.Test.GetDescription()
	sctr.Log = tr.Log
	sctr.CRName = tr.CRName
	sctr.Suggestions = tr.Suggestions
	if sctr.Suggestions == nil {
		sctr.Suggestions = []string{}
	}
	stringErrors := []string{}
	for _, err := range tr.Errors {
		stringErrors = append(stringErrors, err.Error())
	}
	sctr.Errors = stringErrors
	sctr.Labels = tr.Test.GetLabels()
	return sctr
}
