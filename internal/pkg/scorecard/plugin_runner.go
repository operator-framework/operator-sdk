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
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olminstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	k8sInternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
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
	DeployDirOpt              = "deploy-dir"
	OutputFormatOpt           = "output"
	PluginDirOpt              = "plugin-dir"
	JSONOutputFormat          = "json"
	HumanReadableOutputFormat = "human-readable"
)

type internalPlugin int

const (
	basicOperator  internalPlugin = 0
	olmIntegration internalPlugin = 1
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

var internalPluginLogs bytes.Buffer

func setupAndRunPlugin(plugin internalPlugin, pluginViper *viper.Viper) (scapiv1alpha1.ScorecardOutput, error) {
	logCopy := log
	defer func() { log = logCopy }()
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{DisableColors: true})
	internalPluginLogs = bytes.Buffer{}
	log.SetOutput(&internalPluginLogs)
	// use stderr for logging not related to a single suite
	if err := validateScorecardPluginFlags(pluginViper); err != nil {
		return scapiv1alpha1.ScorecardOutput{}, err
	}
	defer func() {
		if err := cleanupScorecard(); err != nil {
			log.SetOutput(&internalPluginLogs)
			log.Errorf("Failed to cleanup resources: (%v)", err)
		}
	}()

	var (
		tmpNamespaceVar string
		err             error
	)
	kubeconfig, tmpNamespaceVar, err = k8sInternal.GetKubeconfigAndNamespace(pluginViper.GetString(KubeconfigOpt))
	if err != nil {
		return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to build the kubeconfig: %v", err)
	}
	if pluginViper.GetString(NamespaceOpt) == "" {
		pluginViper.Set(NamespaceOpt, tmpNamespaceVar)
	}
	scheme := runtime.NewScheme()
	// scheme for client go
	if err := cgoscheme.AddToScheme(scheme); err != nil {
		return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to add client-go scheme to client: (%v)", err)
	}
	// api extensions scheme (CRDs)
	if err := extscheme.AddToScheme(scheme); err != nil {
		return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to add failed to add extensions api scheme to client: (%v)", err)
	}
	// olm api (CSVs)
	if err := olmapiv1alpha1.AddToScheme(scheme); err != nil {
		return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to add failed to add oml api scheme (CSVs) to client: (%v)", err)
	}
	dynamicDecoder = serializer.NewCodecFactory(scheme).UniversalDeserializer()
	// if a user creates a new CRD, we need to be able to reset the rest mapper
	// temporary kubeclient to get a cached discovery
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to get a kubeclient: %v", err)
	}
	cachedDiscoveryClient := cached.NewMemCacheClient(kubeclient.Discovery())
	restMapper = restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()
	runtimeClient, _ = client.New(kubeconfig, client.Options{Scheme: scheme, Mapper: restMapper})

	csv := &olmapiv1alpha1.ClusterServiceVersion{}
	if pluginViper.GetBool(OLMTestsOpt) {
		yamlSpec, err := ioutil.ReadFile(pluginViper.GetString(CSVPathOpt))
		if err != nil {
			return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to read csv: %v", err)
		}
		if err = yaml.Unmarshal(yamlSpec, csv); err != nil {
			return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("error getting ClusterServiceVersion: %v", err)
		}
	}

	// Extract operator manifests from the CSV if olm-deployed is set.
	if pluginViper.GetBool(OlmDeployedOpt) {
		// Get deploymentName from the deployment manifest within the CSV.
		strat, err := (&olminstall.StrategyResolver{}).UnmarshalStrategy(csv.Spec.InstallStrategy)
		if err != nil {
			return scapiv1alpha1.ScorecardOutput{}, err
		}
		stratDep, ok := strat.(*olminstall.StrategyDetailsDeployment)
		if !ok {
			return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("expected StrategyDetailsDeployment, got strategy of type %T", strat)
		}
		deploymentName = stratDep.DeploymentSpecs[0].Name
		// Get the proxy pod, which should have been created with the CSV.
		proxyPodGlobal, err = getPodFromDeployment(deploymentName, pluginViper.GetString(NamespaceOpt))
		if err != nil {
			return scapiv1alpha1.ScorecardOutput{}, err
		}

		// Create a temporary CR manifest from metadata if one is not provided.
		crJSONStr, ok := csv.ObjectMeta.Annotations["alm-examples"]
		if ok && pluginViper.GetString(CRManifestOpt) == "" {
			var crs []interface{}
			if err = json.Unmarshal([]byte(crJSONStr), &crs); err != nil {
				return scapiv1alpha1.ScorecardOutput{}, err
			}
			// TODO: run scorecard against all CR's in CSV.
			cr := crs[0]
			crJSONBytes, err := json.Marshal(cr)
			if err != nil {
				return scapiv1alpha1.ScorecardOutput{}, err
			}
			crYAMLBytes, err := yaml.JSONToYAML(crJSONBytes)
			if err != nil {
				return scapiv1alpha1.ScorecardOutput{}, err
			}
			crFile, err := ioutil.TempFile("", "cr.yaml")
			if err != nil {
				return scapiv1alpha1.ScorecardOutput{}, err
			}
			if _, err := crFile.Write(crYAMLBytes); err != nil {
				return scapiv1alpha1.ScorecardOutput{}, err
			}
			pluginViper.Set(CRManifestOpt, crFile.Name())
			defer func() {
				err := os.Remove(pluginViper.GetString(CRManifestOpt))
				if err != nil {
					log.Errorf("Could not delete temporary CR manifest file: (%v)", err)
				}
			}()
		}

	} else {
		// If no namespaced manifest path is given, combine
		// deploy/{service_account,role.yaml,role_binding,operator}.yaml.
		if pluginViper.GetString(NamespacedManifestOpt) == "" {
			file, err := yamlutil.GenerateCombinedNamespacedManifest(scaffold.DeployDir)
			if err != nil {
				return scapiv1alpha1.ScorecardOutput{}, err
			}
			pluginViper.Set(NamespacedManifestOpt, file.Name())
			defer func() {
				err := os.Remove(pluginViper.GetString(NamespacedManifestOpt))
				if err != nil {
					log.Errorf("Could not delete temporary namespace manifest file: (%v)", err)
				}
				pluginViper.Set(NamespacedManifestOpt, "")
			}()
		}
		// If no global manifest is given, combine all CRD's in the given CRD's dir.
		if pluginViper.GetString(GlobalManifestOpt) == "" {
			if pluginViper.GetString(CRDsDirOpt) == "" {
				pluginViper.Set(CRDsDirOpt, filepath.Join(scaffold.DeployDir, "crds"))
			}
			gMan, err := yamlutil.GenerateCombinedGlobalManifest(pluginViper.GetString(CRDsDirOpt))
			if err != nil {
				return scapiv1alpha1.ScorecardOutput{}, err
			}
			pluginViper.Set(GlobalManifestOpt, gMan.Name())
			defer func() {
				err := os.Remove(pluginViper.GetString(GlobalManifestOpt))
				if err != nil {
					log.Errorf("Could not delete global manifest file: (%v)", err)
				}
				pluginViper.Set(GlobalManifestOpt, "")
			}()
		}
	}

	crs := pluginViper.GetStringSlice(CRManifestOpt)
	// check if there are duplicate CRs
	gvks := []schema.GroupVersionKind{}
	for _, cr := range crs {
		file, err := ioutil.ReadFile(cr)
		if err != nil {
			return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to read file: %s", cr)
		}
		newGVKs, err := getGVKs(file)
		if err != nil {
			return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("could not get GVKs for resource(s) in file: %s, due to error: (%v)", cr, err)
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

	var suites []TestSuite
	for _, cr := range crs {
		logReadWriter = &bytes.Buffer{}
		log.SetOutput(logReadWriter)
		log.Printf("Running for cr: %s", cr)
		if !pluginViper.GetBool(OlmDeployedOpt) {
			if err := createFromYAMLFile(pluginViper.GetString(GlobalManifestOpt)); err != nil {
				return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to create global resources: %v", err)
			}
			if err := createFromYAMLFile(pluginViper.GetString(NamespacedManifestOpt)); err != nil {
				return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to create namespaced resources: %v", err)
			}
		}
		if err := createFromYAMLFile(cr); err != nil {
			return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to create cr resource: %v", err)
		}
		obj, err := yamlToUnstructured(cr)
		if err != nil {
			return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to decode custom resource manifest into object: %s", err)
		}
		if err := waitUntilCRStatusExists(obj); err != nil {
			return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed waiting to check if CR status exists: %v", err)
		}
		if plugin == basicOperator {
			conf := BasicTestConfig{
				Client:   runtimeClient,
				CR:       obj,
				ProxyPod: proxyPodGlobal,
			}
			basicTests := NewBasicTestSuite(conf)
			basicTests.Run(context.TODO())
			logs, err := ioutil.ReadAll(logReadWriter)
			if err != nil {
				basicTests.Log = fmt.Sprintf("failed to read log buffer: %v", err)
			} else {
				basicTests.Log = string(logs)
			}
			suites = append(suites, *basicTests)
		}
		if plugin == olmIntegration {
			conf := OLMTestConfig{
				Client:   runtimeClient,
				CR:       obj,
				CSV:      csv,
				CRDsDir:  pluginViper.GetString(CRDsDirOpt),
				ProxyPod: proxyPodGlobal,
			}
			olmTests := NewOLMTestSuite(conf)
			olmTests.Run(context.TODO())
			logs, err := ioutil.ReadAll(logReadWriter)
			if err != nil {
				olmTests.Log = fmt.Sprintf("failed to read log buffer: %v", err)
			} else {
				olmTests.Log = string(logs)
			}
			suites = append(suites, *olmTests)
		}
		// change logging back to main log
		log.SetOutput(&internalPluginLogs)
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
		return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to merge test suite results: %v", err)
	}
	output := TestSuitesToScorecardOutput(suites, "")
	for idx, suite := range output.Results {
		output.Results[idx] = UpdateSuiteStates(suite)
	}
	return output, nil
}

func validateScorecardPluginFlags(pluginViper *viper.Viper) error {
	if !pluginViper.GetBool(OlmDeployedOpt) && pluginViper.GetStringSlice(CRManifestOpt) == nil {
		return errors.New("cr-manifest config option must be set")
	}
	if !pluginViper.GetBool(BasicTestsOpt) && !pluginViper.GetBool(OLMTestsOpt) {
		return errors.New("at least one test type must be set")
	}
	if pluginViper.GetBool(OLMTestsOpt) && pluginViper.GetString(CSVPathOpt) == "" {
		return fmt.Errorf("csv-path must be set if olm-tests is enabled")
	}
	if pluginViper.GetBool(OlmDeployedOpt) && pluginViper.GetString(CSVPathOpt) == "" {
		return fmt.Errorf("csv-path must be set if olm-deployed is enabled")
	}
	pullPolicy := pluginViper.GetString(ProxyPullPolicyOpt)
	if pullPolicy != "Always" && pullPolicy != "Never" && pullPolicy != "PullIfNotPresent" {
		return fmt.Errorf("invalid proxy pull policy: (%s); valid values: Always, Never, PullIfNotPresent", pullPolicy)
	}
	return nil
}
