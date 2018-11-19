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

package scorecard

import (
	"fmt"
	"io/ioutil"

	k8sInternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"

	olmApi "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: use a config file to reduce number of flags users
// have to provide for the config

// Config stores all scorecard config passed as flags
type Config struct {
	Namespace      string
	KubeconfigPath string
	InitTimeout    int
	CSVPath        string
	BasicTests     bool
	OlmTests       bool
	NamespacedMan  string
	GlobalMan      string
	CrMan          string
	Verbose        bool
}

var SCConf Config

const (
	basicOperator  = "Basic Operator"
	olmIntegration = "OLM Integration"
	goodTenant     = "Good Tentant"
)

// TODO: add point weights to tests
type scorecardTest struct {
	testType      string
	name          string
	description   string
	earnedPoints  int
	maximumPoints int
}

type cleanupFn func() error

var (
	apiversion     string
	kind           string
	name           string
	kubeconfig     *rest.Config
	scTests        []scorecardTest
	dynamicDecoder runtime.Decoder
	runtimeClient  client.Client
	restMapper     *restmapper.DeferredDiscoveryRESTMapper
	deploymentName string
	proxyPod       *v1.Pod
	cleanupFns     []cleanupFn
)

const scorecardPodName = "operator-scorecard-test"

func ScorecardTests(cmd *cobra.Command, args []string) error {
	// in main.go, we catch and print errors, so we don't want cobra to print the error itself
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if SCConf.Verbose {
		log.SetLevel(log.DebugLevel)
	}
	defer cleanupScorecard()
	var err error
	kubeconfig, SCConf.Namespace, err = k8sInternal.GetKubeconfigAndNamespace(SCConf.KubeconfigPath)
	if err != nil {
		return err
	}
	scheme := runtime.NewScheme()
	// scheme for client go
	cgoscheme.AddToScheme(scheme)
	// api extensions scheme (CRDs)
	extscheme.AddToScheme(scheme)
	// olm api (CSVs)
	olmApi.AddToScheme(scheme)
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
	err = createFromYAMLFile(SCConf.GlobalMan, false)
	if err != nil {
		return fmt.Errorf("failed to create global resources: %v", err)
	}
	err = createFromYAMLFile(SCConf.NamespacedMan, false)
	if err != nil {
		return fmt.Errorf("failed to create namespaced resources: %v", err)
	}
	err = createFromYAMLFile(SCConf.CrMan, true)
	if err != nil {
		return fmt.Errorf("failed to create cr resource: %v", err)
	}
	obj := unstructured.Unstructured{}
	obj.SetAPIVersion(apiversion)
	obj.SetKind(kind)
	if SCConf.BasicTests {
		fmt.Printf("Checking spec and stat\n")
		err = checkSpecAndStat(runtimeClient, obj)
		if err != nil {
			return err
		}
		fmt.Printf("Checking status update\n")
		err = checkStatusUpdate(runtimeClient, obj)
		if err != nil {
			return err
		}
		logs, err := writingIntoCRsHasEffect(obj)
		if err != nil {
			return err
		}
		log.Debugf("Scorecard Proxy Logs: %v\n", logs)
	}
	if SCConf.OlmTests {
		yamlSpec, err := ioutil.ReadFile(SCConf.CSVPath)
		if err != nil {
			return fmt.Errorf("failed to read csv: %v", err)
		}
		rawCSV, _, err := dynamicDecoder.Decode(yamlSpec, nil, nil)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		csv := &olmApi.ClusterServiceVersion{}
		switch o := rawCSV.(type) {
		case *olmApi.ClusterServiceVersion:
			csv = o
		default:
			return fmt.Errorf("provided yaml file not of ClusterServiceVersion type")
		}
		fmt.Printf("Checking for CRD resources\n")
		crdsHaveResources(csv)
		fmt.Printf("Checking for existence CR example\n")
		annotationsContainExamples(csv)
		fmt.Printf("Checking spec descriptors\n")
		err = specDescriptors(csv, runtimeClient, obj)
		if err != nil {
			return err
		}
		fmt.Printf("Checking status descriptors\n")
		err = statusDescriptors(csv, runtimeClient, obj)
		if err != nil {
			return err
		}
	}
	var totalEarned, totalMax int
	for _, testType := range []string{basicOperator, olmIntegration, goodTenant} {
		fmt.Printf("%s:\n", testType)
		for _, test := range scTests {
			if test.testType == testType {
				fmt.Printf("\t%s: %d/%d points\n", test.name, test.earnedPoints, test.maximumPoints)
				totalEarned += test.earnedPoints
				totalMax += test.maximumPoints
			}
		}
	}
	fmt.Printf("\nTotal Score: %d/%d points\n", totalEarned, totalMax)
	return nil
}
