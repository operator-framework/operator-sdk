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
	"os"
	"time"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	testNamespace     string
	kubeconfigCluster string
	imagePullPolicy   bool
)

func NewTestClusterCmd() *cobra.Command {
	testCmd := &cobra.Command{
		Use:   "cluster <image name> [flags]",
		Short: "Run End-To-End tests using image with embedded test binary",
		RunE:  testClusterFunc,
	}
	defaultKubeConfig := ""
	homedir, ok := os.LookupEnv("HOME")
	if ok {
		defaultKubeConfig = homedir + "/.kube/config"
	}
	testCmd.Flags().StringVarP(&testNamespace, "namespace", "n", "default", "Namespace to run tests in")
	testCmd.Flags().StringVarP(&kubeconfigCluster, "kubeconfig", "k", defaultKubeConfig, "Kubeconfig path")
	testCmd.Flags().BoolVarP(&imagePullPolicy, "imagePullPolicy", "i", false, "Set test pod image pull policy to 'Never'")

	return testCmd
}

func testClusterFunc(cmd *cobra.Command, args []string) error {
	// in main.go, we catch and print errors, so we don't want cobra to print the error itself
	cmd.SilenceErrors = true
	if len(args) != 1 {
		return fmt.Errorf("operator-sdk test cluster requires exactly 1 argument")
	}
	// cobra prints its help message on error; we silence that here because any errors below
	// are due to the test failing, not incorrect user input
	cmd.SilenceUsage = true
	pullPolicy := v1.PullAlways
	if imagePullPolicy {
		pullPolicy = v1.PullNever
	}
	testPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "operator-test",
		},
		Spec: v1.PodSpec{
			RestartPolicy: v1.RestartPolicyNever,
			Containers: []v1.Container{{
				Name:            "operator-test",
				Image:           args[0],
				ImagePullPolicy: pullPolicy,
				Command:         []string{"/go-test.sh"},
				Env: []v1.EnvVar{{
					Name:      "TEST_NAMESPACE",
					ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
				}},
			}},
		},
	}
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigCluster)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %v", err)
	}
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubeclient: %v", err)
	}
	testPod, err = kubeclient.CoreV1().Pods(testNamespace).Create(testPod)
	if err != nil {
		return fmt.Errorf("failed to create test pod: %v", err)
	}
	defer func() {
		err = kubeclient.CoreV1().Pods(testNamespace).Delete(testPod.Name, &metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf("Warning: failed to delete test pod")
		}
	}()
	for {
		testPod, err = kubeclient.CoreV1().Pods(testNamespace).Get(testPod.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get test pod: %v", err)
		}
		if testPod.Status.Phase != v1.PodSucceeded && testPod.Status.Phase != v1.PodFailed {
			time.Sleep(time.Second * 5)
			continue
		} else if testPod.Status.Phase == v1.PodSucceeded {
			fmt.Printf("Test Successfully Completed\n")
			return nil
		} else if testPod.Status.Phase == v1.PodFailed {
			req := kubeclient.CoreV1().Pods(testNamespace).GetLogs(testPod.Name, &v1.PodLogOptions{})
			readCloser, err := req.Stream()
			if err != nil {
				return fmt.Errorf("test failed and failed to get error logs")
			}
			defer readCloser.Close()
			buf := new(bytes.Buffer)
			buf.ReadFrom(readCloser)
			return fmt.Errorf("test failed:\n%s", buf.String())
		}
	}
}
