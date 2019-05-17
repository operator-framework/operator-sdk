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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	k8sInternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/config"
	scinternal "github.com/operator-framework/operator-sdk/pkg/scorecard/internal"

	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olminstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	basicOperator  = "Basic Operator"
	olmIntegration = "OLM Integration"
)

// TODO: parameterize the ReadWriter.
func (c *ScorecardCmd) ConfigureLogger() error {
	outputFmt := viper.GetString(OutputFormatOpt)
	if outputFmt == HumanReadableOutputFormat {
		c.logReadWriter = os.Stdout
	} else if outputFmt == JSONOutputFormat {
		c.logReadWriter = &bytes.Buffer{}
	} else {
		return fmt.Errorf("invalid output format: %s", outputFmt)
	}
	c.log = logrus.New()
	c.log.SetOutput(c.logReadWriter)
	return nil
}

func (c *ScorecardCmd) Run() error {
	if c.log == nil {
		return fmt.Errorf("scorecard logger must be set with ConfigureLogger before invoking Run")
	}

	if err := c.validateFlags(); err != nil {
		return err
	}
	if err := c.setInGlobal(); err != nil {
		return err
	}

	rc := &scinternal.ResourceConfig{
		Namespace:       viper.GetString(NamespaceOpt),
		ProxyImage:      viper.GetString(ProxyImageOpt),
		ProxyPullPolicy: viper.GetString(ProxyPullPolicyOpt),
		InitTimeout:     viper.GetInt(InitTimeoutOpt),
	}
	rc.SetLogger(c.log)

	defer func() {
		if err := rc.Cleanup(); err != nil {
			c.log.Errorf("Failed to cleanup resources: (%v)", err)
		}
	}()

	var (
		tmpNamespaceVar string
		err             error
	)
	rc.Kubeconfig, tmpNamespaceVar, err = k8sInternal.GetKubeconfigAndNamespace(viper.GetString(KubeconfigPathOpt))
	if err != nil {
		return fmt.Errorf("failed to build the kubeconfig: %v", err)
	}
	if rc.Namespace == "" {
		rc.Namespace = tmpNamespaceVar
	}
	scheme := runtime.NewScheme()
	// scheme for client go
	if err := cgoscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add client-go scheme to client: (%v)", err)
	}
	// api extensions scheme (CRDs)
	if err := extscheme.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add failed to add extensions api scheme to client: (%v)", err)
	}
	// olm api (CS
	if err := olmapiv1alpha1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add failed to add oml api scheme (CSVs) to client: (%v)", err)
	}
	rc.Decoder = serializer.NewCodecFactory(scheme).UniversalDeserializer()
	// if a user creates a new CRD, we need to be able to reset the rest mapper
	// temporary kubeclient to get a cached discovery
	kubeclient, err := kubernetes.NewForConfig(rc.Kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get a kubeclient: %v", err)
	}
	cachedDiscoveryClient := cached.NewMemCacheClient(kubeclient.Discovery())
	rc.RestMapper = restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	rc.RestMapper.Reset()
	rc.Client, _ = client.New(rc.Kubeconfig, client.Options{Scheme: scheme, Mapper: rc.RestMapper})

	csv := &olmapiv1alpha1.ClusterServiceVersion{}
	if viper.GetBool(OLMTestsOpt) {
		yamlSpec, err := ioutil.ReadFile(viper.GetString(CSVPathOpt))
		if err != nil {
			return fmt.Errorf("failed to read csv: %v", err)
		}
		if err = yaml.Unmarshal(yamlSpec, csv); err != nil {
			return fmt.Errorf("error getting ClusterServiceVersion: %v", err)
		}
	}

	// Extract operator manifests from the CSV if olm-deployed is set.
	if viper.GetBool(OLMDeployedOpt) {
		// Get deploymentName from the deployment manifest within the CSV.
		strat, err := (&olminstall.StrategyResolver{}).UnmarshalStrategy(csv.Spec.InstallStrategy)
		if err != nil {
			return err
		}
		stratDep, ok := strat.(*olminstall.StrategyDetailsDeployment)
		if !ok {
			return fmt.Errorf("expected StrategyDetailsDeployment, got strategy of type %T", strat)
		}
		rc.DeploymentName = stratDep.DeploymentSpecs[0].Name
		// Get the proxy pod, which should have been created with the CSV.
		if rc.SetProxyPod(); err != nil {
			return err
		}

		// Create a temporary CR manifest from metadata if one is not provided.
		crJSONStr, ok := csv.ObjectMeta.Annotations["alm-examples"]
		if ok && len(viper.GetStringSlice(CRManifestOpt)) == 0 {
			var crs []interface{}
			if err = json.Unmarshal([]byte(crJSONStr), &crs); err != nil {
				return err
			}
			// TODO: run scorecard against all CR's in CSV.
			cr := crs[0]
			crJSONBytes, err := json.Marshal(cr)
			if err != nil {
				return err
			}
			crYAMLBytes, err := yaml.JSONToYAML(crJSONBytes)
			if err != nil {
				return err
			}
			crFile, err := ioutil.TempFile("", "cr.yaml")
			if err != nil {
				return err
			}
			if _, err := crFile.Write(crYAMLBytes); err != nil {
				return err
			}
			viper.Set(CRManifestOpt, []string{crFile.Name()})
			defer func() {
				err := os.RemoveAll(crFile.Name())
				if err != nil {
					c.log.Errorf("Could not delete temporary CR manifest file: (%v)", err)
				}
			}()
		}

	} else {
		// If no namespaced manifest path is given, combine
		// deploy/{service_account,role.yaml,role_binding,operator}.yaml.
		if viper.GetString(NamespacedManifestOpt) == "" {
			nMan, err := yamlutil.GenerateCombinedNamespacedManifest(viper.GetString(config.DeployDirOpt))
			if err != nil {
				return err
			}
			viper.Set(NamespacedManifestOpt, nMan.Name())
			defer func() {
				err := os.Remove(nMan.Name())
				if err != nil {
					c.log.Errorf("Could not delete temporary namespace manifest file: (%v)", err)
				}
			}()
		}
		// If no global manifest is given, combine all CRD's in the given CRD's dir.
		if viper.GetString(GlobalManifestOpt) == "" {
			gMan, err := yamlutil.GenerateCombinedGlobalManifest(viper.GetString(config.CRDsDirOpt))
			if err != nil {
				return err
			}
			viper.Set(GlobalManifestOpt, gMan.Name())
			defer func() {
				err := os.Remove(gMan.Name())
				if err != nil {
					c.log.Errorf("Could not delete global manifest file: (%v)", err)
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
			c.log.Warnf("Duplicate gvks in CR list detected (%s); results may be inaccurate", gvk)
		}
		dupMap[gvk] = true
	}

	var pluginResults []scapiv1alpha1.ScorecardOutput
	var suites []scinternal.TestSuite
	for _, cr := range crs {
		// TODO: Change built-in tests into plugins
		// Run built-in tests.
		fmt.Printf("Running for cr: %s\n", cr)
		if !viper.GetBool(OLMDeployedOpt) {
			if err := rc.CreateFromYAMLFile(viper.GetString(GlobalManifestOpt)); err != nil {
				return fmt.Errorf("failed to create global resources: %v", err)
			}
			if err := rc.CreateFromYAMLFile(viper.GetString(NamespacedManifestOpt)); err != nil {
				return fmt.Errorf("failed to create namespaced resources: %v", err)
			}
		}
		if err := rc.CreateFromYAMLFile(cr); err != nil {
			return fmt.Errorf("failed to create cr resource: %v", err)
		}
		obj, err := scinternal.YamlToUnstructured(cr)
		if err != nil {
			return fmt.Errorf("failed to decode custom resource manifest into object: %s", err)
		}
		// set obj's namespace
		obj.SetNamespace(rc.Namespace)
		if err := rc.WaitUntilCRStatusExists(obj); err != nil {
			return fmt.Errorf("failed waiting to check if CR status exists: %v", err)
		}
		if viper.GetBool(BasicTestsOpt) {
			conf := scinternal.BasicTestConfig{
				ResourceConfig: rc,
				CR:             obj,
			}
			basicTests := scinternal.NewBasicTestSuite(conf)
			basicTests.Run(context.TODO())
			suites = append(suites, *basicTests)
		}
		if viper.GetBool(OLMTestsOpt) {
			conf := scinternal.OLMTestConfig{
				ResourceConfig: rc,
				CR:             obj,
				CSV:            csv,
				CRDsDir:        viper.GetString(config.CRDsDirOpt),
			}
			olmTests := scinternal.NewOLMTestSuite(conf)
			olmTests.Run(context.TODO())
			suites = append(suites, *olmTests)
		}
		// set up clean environment for every CR
		if err := rc.Cleanup(); err != nil {
			c.log.Errorf("Failed to cleanup resources: (%v)", err)
		}

		// reset cleanup functions
		rc.CleanupFns = scinternal.CleanupFns{}
		// clear name of operator deployment
		rc.DeploymentName = ""
	}
	suites, err = scinternal.MergeSuites(suites)
	if err != nil {
		return fmt.Errorf("failed to merge test suite results: %v", err)
	}

	for _, suite := range suites {
		// convert to ScorecardOutput format
		// will add log when basic and olm tests are separated into plugins
		pluginResults = append(pluginResults, scinternal.TestSuitesToScorecardOutput([]scinternal.TestSuite{suite}, ""))
	}
	// Run plugins
	pluginDir := viper.GetString(PluginDirOpt)
	if dir, err := os.Stat(pluginDir); err != nil || !dir.IsDir() {
		c.log.Warnf("Plugin directory not found; skipping plugin tests: %v", err)
		return c.printOutput(pluginResults)
	}
	if err := os.Chdir(pluginDir); err != nil {
		return fmt.Errorf("failed to chdir into scorecard plugin directory: %v", err)
	}
	// executable files must be in "bin" subdirectory
	files, err := ioutil.ReadDir("bin")
	if err != nil {
		return fmt.Errorf("failed to list files in %s/bin: %v", pluginDir, err)
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
			c.log.Errorf("Plugin `%s` failed with error (%v)", file.Name(), err)
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
			c.log.Errorf("Output from plugin `%s` failed to unmarshal with error (%v)", file.Name(), err)
			continue
		}
		stderrString := string(stderr.Bytes())
		if len(stderrString) != 0 {
			c.log.Warn(stderrString)
		}
		pluginResults = append(pluginResults, result)
	}
	return c.printOutput(pluginResults)
}

func (c *ScorecardCmd) printOutput(pluginOutputs []scapiv1alpha1.ScorecardOutput) error {
	totalScore := 0.0
	// Update the state for the tests
	for _, suite := range pluginOutputs {
		for idx, res := range suite.Results {
			suite.Results[idx] = scinternal.UpdateSuiteStates(res)
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
		log, err := ioutil.ReadAll(c.logReadWriter)
		if err != nil {
			return fmt.Errorf("failed to read log buffer: %v", err)
		}
		scTest := scinternal.CombineScorecardOutput(pluginOutputs, string(log))
		// Pretty print so users can also read the json output
		bytes, err := json.MarshalIndent(scTest, "", "  ")
		if err != nil {
			return err
		}
		fmt.Printf("%s\n", string(bytes))
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
