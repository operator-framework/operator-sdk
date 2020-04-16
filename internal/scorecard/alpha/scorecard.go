// Copyright 2020 The Operator-SDK Authors
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

package alpha

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/operator-framework/operator-sdk/version"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Options struct {
	Config       Config
	Selector     labels.Selector
	List         bool
	OutputFormat string
	Kubeconfig   string
	Client       kubernetes.Interface
}

// RunTests executes the scorecard tests as configured
func RunTests(o Options) error {
	tests := selectTests(o.Selector, o.Config.Tests)

	for i := 0; i < len(tests); i++ {
		if err := runTest(o, tests[i]); err != nil {
			return fmt.Errorf("test %s failed %s", tests[i].Name, err.Error())
		}
	}

	return nil
}

// LoadConfig will find and return the scorecard config, the config file
// can be passed in via command line flag or from a bundle location or
// bundle image
func LoadConfig(configFilePath string) (Config, error) {
	c := Config{}

	// TODO handle getting config from bundle (ondisk or image)
	yamlFile, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		return c, err
	}

	if err := yaml.Unmarshal(yamlFile, &c); err != nil {
		return c, err
	}

	return c, nil
}

// selectTests applies an optionally passed selector expression
// against the configured set of tests, returning the selected tests
func selectTests(selector labels.Selector, tests []ScorecardTest) []ScorecardTest {

	selected := make([]ScorecardTest, 0)
	for i := 0; i < len(tests); i++ {
		if selector.String() == "" || selector.Matches(labels.Set(tests[i].Labels)) {
			// TODO olm manifests check
			selected = append(selected, tests[i])
		}
	}
	return selected
}

// runTest executes a single test
// TODO once tests exists, handle the test output
func runTest(o Options, test ScorecardTest) error {
	if test.Name == "" {
		return errors.New("todo - remove later, only for linter")
	}
	log.Printf("running test %s labels %v", test.Name, test.Labels)

	// Create a Pod to run the test

	podDef := getPodDefinition(test)
	_, err := o.Client.CoreV1().Pods("default").Create(podDef)
	return err
}

func ConfigDocLink() string {
	if strings.HasSuffix(version.Version, "+git") {
		return "https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/scorecard.md"
	}
	return fmt.Sprintf(
		"https://github.com/operator-framework/operator-sdk/blob/%s/doc/test-framework/scorecard.md",
		version.Version)
}

// GetKubeClient will get a kubernetes client from the ...
func GetKubeClient(kubeconfig string) (client kubernetes.Interface, err error) {

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return client, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return client, err
	}

	return clientset, err
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func getPodDefinition(test ScorecardTest) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scorecard-test",
			Namespace: "default",
			Labels: map[string]string{
				"name": "scorecard-test",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:            "scorecard-test",
					Image:           "quay.io/operator-framework/scorecard-test:dev",
					ImagePullPolicy: v1.PullIfNotPresent,
					Command: []string{
						"/usr/local/bin/scorecard-test",
					},
					Args: []string{
						test.Entrypoint,
					},
					VolumeMounts: []v1.VolumeMount{
						{
							MountPath: "/scorecard",
							Name:      "scorecard-bundle",
							ReadOnly:  true,
						},
					},
				},
			},
			Volumes: []v1.Volume{
				{
					Name: "scorecard-bundle",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: "scorecard-bundle",
							},
						},
					},
				},
			},
		},
	}
}
