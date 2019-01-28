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
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	k8sInternal "github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
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
	Namespace          string
	KubeconfigPath     string
	InitTimeout        int
	CSVPath            string
	BasicTests         bool
	OLMTests           bool
	TenantTests        bool
	NamespacedManifest string
	GlobalManifest     string
	CRManifest         string
	ProxyImage         string
	ProxyPullPolicy    string
	Verbose            bool
}

var SCConf Config

const (
	basicOperator  = "Basic Operator"
	olmIntegration = "OLM Integration"
	goodTenant     = "Good Tenant"
)

var (
	kubeconfig     *rest.Config
	scSuggestions  []string
	dynamicDecoder runtime.Decoder
	runtimeClient  client.Client
	restMapper     *restmapper.DeferredDiscoveryRESTMapper
	deploymentName string
	proxyPod       *v1.Pod
	cleanupFns     []cleanupFn
)

const scorecardPodName = "operator-scorecard-test"

func ScorecardTests(cmd *cobra.Command, args []string) error {
	if !SCConf.BasicTests && !SCConf.OLMTests {
		return errors.New("at least one test type is required")
	}
	if SCConf.OLMTests && SCConf.CSVPath == "" {
		return fmt.Errorf("if olm-tests is enabled, the --csv-path flag must be set")
	}
	if SCConf.ProxyPullPolicy != "Always" && SCConf.ProxyPullPolicy != "Never" && SCConf.ProxyPullPolicy != "PullIfNotPresent" {
		return fmt.Errorf("invalid proxy pull policy: (%s); valid values: Always, Never", SCConf.ProxyPullPolicy)
	}
	cmd.SilenceUsage = true
	if SCConf.Verbose {
		log.SetLevel(log.DebugLevel)
	}
	// if no namespaced manifest path is given, combine deploy/service_account.yaml, deploy/role.yaml, deploy/role_binding.yaml and deploy/operator.yaml
	if SCConf.NamespacedManifest == "" {
		file, err := yamlutil.GenerateCombinedNamespacedManifest()
		if err != nil {
			log.Fatal(err)
		}
		SCConf.NamespacedManifest = file.Name()
		defer func() {
			err := os.Remove(SCConf.NamespacedManifest)
			if err != nil {
				log.Fatalf("Could not delete temporary namespace manifest file: (%v)", err)
			}
		}()
	}
	if SCConf.GlobalManifest == "" {
		file, err := yamlutil.GenerateCombinedGlobalManifest()
		if err != nil {
			log.Fatal(err)
		}
		SCConf.GlobalManifest = file.Name()
		defer func() {
			err := os.Remove(SCConf.GlobalManifest)
			if err != nil {
				log.Fatalf("Could not delete global manifest file: (%v)", err)
			}
		}()
	}
	defer func() {
		if err := cleanupScorecard(); err != nil {
			log.Errorf("Failed to clenup resources: (%v)", err)
		}
	}()
	var err error
	kubeconfig, SCConf.Namespace, err = k8sInternal.GetKubeconfigAndNamespace(SCConf.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to build the kubeconfig: %v", err)
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
	if err := createFromYAMLFile(SCConf.GlobalManifest); err != nil {
		return fmt.Errorf("failed to create global resources: %v", err)
	}
	if err := createFromYAMLFile(SCConf.NamespacedManifest); err != nil {
		return fmt.Errorf("failed to create namespaced resources: %v", err)
	}
	if err := createFromYAMLFile(SCConf.CRManifest); err != nil {
		return fmt.Errorf("failed to create cr resource: %v", err)
	}
	obj, err := yamlToUnstructured(SCConf.CRManifest)
	if err != nil {
		return fmt.Errorf("failed to decode custom resource manifest into object: %s", err)
	}
	vars := ScorecardVars{
		client:        &runtimeClient,
		crObj:         obj,
		proxyPod:      proxyPod,
		timeout:       60,
		retryInterval: 5,
	}
	if err := waitUntilReady(obj); err != nil {
		return fmt.Errorf("failed waiting for CR to be ready: %v", err)
	}
	if SCConf.BasicTests {
		for _, test := range BasicTests.tests {
			err = test.execute(vars)
			if err != nil {
				return err
			}
		}
	}
	if SCConf.OLMTests {
		yamlSpec, err := ioutil.ReadFile(SCConf.CSVPath)
		if err != nil {
			return fmt.Errorf("failed to read csv: %v", err)
		}
		rawCSV, _, err := dynamicDecoder.Decode(yamlSpec, nil, nil)
		if err != nil {
			return err
		}
		csv := &olmapiv1alpha1.ClusterServiceVersion{}
		switch o := rawCSV.(type) {
		case *olmapiv1alpha1.ClusterServiceVersion:
			csv = o
		default:
			return fmt.Errorf("provided yaml file not of ClusterServiceVersion type")
		}
		vars.csvObj = csv
		for _, test := range OLMTests.tests {
			err = test.execute(vars)
			if err != nil {
				return err
			}
		}
	}
	var totalScores []int
	if SCConf.BasicTests {
		fmt.Println("Basic Tests")
		for _, test := range BasicTests.tests {
			if !(test.scores[0].earnedPoints == 0 && test.scores[0].maximumPoints == 0) {
				fmt.Printf("\t%s: %d/%d points\n", test.name, test.scores[0].earnedPoints, test.scores[0].maximumPoints)
			} else {
				fmt.Printf("\t%s: N/A (depends on an earlier test that failed)\n", test.name)
			}
		}
		totalScore := BasicTests.calculateTotalScore()
		fmt.Printf("Basic Tests Score: %d%%\n\n", totalScore)
		totalScores = append(totalScores, totalScore)
	}
	if SCConf.OLMTests {
		fmt.Println("OLM Tests")
		for _, test := range OLMTests.tests {
			if !(test.scores[0].earnedPoints == 0 && test.scores[0].maximumPoints == 0) {
				fmt.Printf("\t%s: %d/%d points\n", test.name, test.scores[0].earnedPoints, test.scores[0].maximumPoints)
			} else {
				fmt.Printf("\t%s: N/A (depends on an earlier test that failed)\n", test.name)
			}
		}
		totalScore := OLMTests.calculateTotalScore()
		fmt.Printf("OLM Tests Score: %d%%\n\n", totalScore)
		totalScores = append(totalScores, totalScore)
	}
	/* not yet implemented
	if SCConf.TenantTests {
	*/
	totalScore := 0.0
	for _, score := range totalScores {
		totalScore += float64(score)
	}
	totalScore = totalScore / float64(len(totalScores))
	fmt.Printf("\nTotal Score: %.0f%%\n", totalScore)
	for _, suggestion := range scSuggestions {
		// 33 is yellow (specifically, the same shade of yellow that logrus uses for warnings)
		fmt.Printf("\x1b[%dmSUGGESTION:\x1b[0m %s\n", 33, suggestion)
	}
	return nil
}
