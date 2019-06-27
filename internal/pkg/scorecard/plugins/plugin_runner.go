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
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	schelpers "github.com/operator-framework/operator-sdk/internal/pkg/scorecard/helpers"
	k8sInternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"
	scapiv1alpha1 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha1"

	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olminstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	"github.com/pkg/errors"
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
	scorecardPodName       = "operator-scorecard-test"
	scorecardContainerName = "scorecard-proxy"
)

var log *logrus.Logger

func RunInternalPlugin(plugin PluginType, config *viper.Viper, logFile io.ReadWriter) (scapiv1alpha1.ScorecardOutput, error) {
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{DisableColors: true})
	log.SetOutput(logFile)
	// use stderr for logging not related to a single suite
	if err := validateScorecardPluginFlags(config); err != nil {
		return scapiv1alpha1.ScorecardOutput{}, err
	}
	defer func() {
		if err := cleanupScorecard(); err != nil {
			log.SetOutput(logFile)
			log.Errorf("Failed to cleanup resources: (%v)", err)
		}
	}()

	var (
		tmpNamespaceVar string
		err             error
	)
	kubeconfig, tmpNamespaceVar, err = k8sInternal.GetKubeconfigAndNamespace(config.GetString(KubeconfigOpt))
	if err != nil {
		return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to build the kubeconfig: %v", err)
	}
	if config.GetString(NamespaceOpt) == "" {
		config.Set(NamespaceOpt, tmpNamespaceVar)
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
	if config.GetBool(OLMTestsOpt) {
		yamlSpec, err := ioutil.ReadFile(config.GetString(CSVPathOpt))
		if err != nil {
			return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to read csv: %v", err)
		}
		if err = yaml.Unmarshal(yamlSpec, csv); err != nil {
			return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("error getting ClusterServiceVersion: %v", err)
		}
	}

	// Extract operator manifests from the CSV if olm-deployed is set.
	if config.GetBool(OlmDeployedOpt) {
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
		proxyPodGlobal, err = getPodFromDeployment(deploymentName, config.GetString(NamespaceOpt))
		if err != nil {
			return scapiv1alpha1.ScorecardOutput{}, err
		}

		logCRMsg := false
		if crMans := config.GetStringSlice(CRManifestOpt); len(crMans) == 0 {
			// Create a temporary CR manifest from metadata if one is not provided.
			if crJSONStr, ok := csv.ObjectMeta.Annotations["alm-examples"]; ok {
				var crs []interface{}
				if err = json.Unmarshal([]byte(crJSONStr), &crs); err != nil {
					return scapiv1alpha1.ScorecardOutput{}, err
				}
				// TODO: run scorecard against all CR's in CSV.
				cr := crs[0]
				logCRMsg = len(crs) > 1
				crJSONBytes, err := json.Marshal(cr)
				if err != nil {
					return scapiv1alpha1.ScorecardOutput{}, err
				}
				crYAMLBytes, err := yaml.JSONToYAML(crJSONBytes)
				if err != nil {
					return scapiv1alpha1.ScorecardOutput{}, err
				}
				crFile, err := ioutil.TempFile("", "*.cr.yaml")
				if err != nil {
					return scapiv1alpha1.ScorecardOutput{}, err
				}
				if _, err := crFile.Write(crYAMLBytes); err != nil {
					return scapiv1alpha1.ScorecardOutput{}, err
				}
				config.Set(CRManifestOpt, []string{crFile.Name()})
				defer func() {
					for _, f := range config.GetStringSlice(CRManifestOpt) {
						if err := os.Remove(f); err != nil {
							log.Errorf("Could not delete temporary CR manifest file: (%v)", err)
						}
					}
				}()
			} else {
				return scapiv1alpha1.ScorecardOutput{}, errors.New("cr-manifest config option must be set if CSV has no metadata.annotations['alm-examples']")
			}
		} else {
			// TODO: run scorecard against all CR's in CSV.
			config.Set(CRManifestOpt, []string{crMans[0]})
			logCRMsg = len(crMans) > 1
		}
		// Let users know that only the first CR is being tested.
		if logCRMsg {
			log.Infof("The scorecard does not support testing multiple CR's at once when run with --olm-deployed. Testing the first CR %s", config.GetStringSlice(CRManifestOpt)[0])
		}

	} else {
		// If no namespaced manifest path is given, combine
		// deploy/{service_account,role.yaml,role_binding,operator}.yaml.
		if config.GetString(NamespacedManifestOpt) == "" {
			file, err := yamlutil.GenerateCombinedNamespacedManifest(scaffold.DeployDir)
			if err != nil {
				return scapiv1alpha1.ScorecardOutput{}, err
			}
			config.Set(NamespacedManifestOpt, file.Name())
			defer func() {
				err := os.Remove(config.GetString(NamespacedManifestOpt))
				if err != nil {
					log.Errorf("Could not delete temporary namespace manifest file: (%v)", err)
				}
				config.Set(NamespacedManifestOpt, "")
			}()
		}
		// If no global manifest is given, combine all CRD's in the given CRD's dir.
		if config.GetString(GlobalManifestOpt) == "" {
			if config.GetString(CRDsDirOpt) == "" {
				config.Set(CRDsDirOpt, filepath.Join(scaffold.DeployDir, "crds"))
			}
			gMan, err := yamlutil.GenerateCombinedGlobalManifest(config.GetString(CRDsDirOpt))
			if err != nil {
				return scapiv1alpha1.ScorecardOutput{}, err
			}
			config.Set(GlobalManifestOpt, gMan.Name())
			defer func() {
				err := os.Remove(config.GetString(GlobalManifestOpt))
				if err != nil {
					log.Errorf("Could not delete global manifest file: (%v)", err)
				}
				config.Set(GlobalManifestOpt, "")
			}()
		}
	}

	crs := config.GetStringSlice(CRManifestOpt)
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

	var suites []schelpers.TestSuite
	for _, cr := range crs {
		logReadWriter := &bytes.Buffer{}
		log.SetOutput(logReadWriter)
		log.Printf("Running for cr: %s", cr)
		if !config.GetBool(OlmDeployedOpt) {
			if err := createFromYAMLFile(config.GetString(GlobalManifestOpt)); err != nil {
				return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to create global resources: %v", err)
			}
			if err := createFromYAMLFile(config.GetString(NamespacedManifestOpt)); err != nil {
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
		if plugin == BasicOperator {
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
		if plugin == OLMIntegration {
			conf := OLMTestConfig{
				Client:   runtimeClient,
				CR:       obj,
				CSV:      csv,
				CRDsDir:  config.GetString(CRDsDirOpt),
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
		log.SetOutput(logFile)
		// set up clean environment for every CR
		if err := cleanupScorecard(); err != nil {
			log.Errorf("Failed to cleanup resources: (%v)", err)
		}
		// reset cleanup functions
		cleanupFns = []cleanupFn{}
		// clear name of operator deployment
		deploymentName = ""
	}
	suites, err = schelpers.MergeSuites(suites)
	if err != nil {
		return scapiv1alpha1.ScorecardOutput{}, fmt.Errorf("failed to merge test suite results: %v", err)
	}
	output := schelpers.TestSuitesToScorecardOutput(suites, "")
	for idx, suite := range output.Results {
		output.Results[idx] = schelpers.UpdateSuiteStates(suite)
	}
	return output, nil
}
