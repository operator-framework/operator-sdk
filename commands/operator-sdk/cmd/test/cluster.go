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
	"fmt"
	"os"
	"os/exec"
	"time"

	cmdError "github.com/operator-framework/operator-sdk/commands/operator-sdk/error"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	testNamespace             string
	kubeconfigCluster         string
	globalManifestPathCluster string
)

func NewTestClusterCmd() *cobra.Command {
	testCmd := &cobra.Command{
		Use:   "test --test-location <path to tests directory> [flags]",
		Short: "Run End-To-End tests",
		Run:   testClusterFunc,
	}
	defaultKubeConfig := ""
	homedir, ok := os.LookupEnv("HOME")
	if ok {
		defaultKubeConfig = homedir + "/.kube/config"
	}
	testCmd.Flags().StringVarP(&testNamespace, "namespace", "n", "default", "Namespace to run tests in")
	testCmd.Flags().StringVarP(&kubeconfigCluster, "kubeconfig", "k", defaultKubeConfig, "Kubeconfig path")
	testCmd.Flags().StringVarP(&globalManifestPathCluster, "global-init", "g", "", "Path to manifest for Global resources (e.g. CRD manifest)")

	return testCmd
}

func testClusterFunc(cmd *cobra.Command, args []string) {
	if globalManifestPathCluster != "" {
		globalCmd := exec.Command("kubectl", "create", "-f", globalManifestPathCluster)
		cmdOut, err := globalCmd.CombinedOutput()
		if err != nil {
			cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("Could not create global resources: %v\nKubectl Output: %v", err, cmdOut))
		}
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
				ImagePullPolicy: v1.PullAlways,
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
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to get kubeconfig: %v", err))
	}
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to create kubeclient: %v", err))
	}
	testPod, err = kubeclient.CoreV1().Pods(testNamespace).Create(testPod)
	if err != nil {
		cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to create test pod: %v", err))
	}
	for {
		testPod, err = kubeclient.CoreV1().Pods(testNamespace).Get("operator-test", metav1.GetOptions{})
		if err != nil {
			cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("failed to get test pod: %v", err))
		}
		if testPod.Status.Phase != v1.PodSucceeded && testPod.Status.Phase != v1.PodFailed {
			time.Sleep(time.Second * 5)
			continue
		} else if testPod.Status.Phase == v1.PodSucceeded {
			fmt.Printf("Test Successfully Completed")
			return
		} else if testPod.Status.Phase == v1.PodFailed {
			cmdError.ExitWithError(cmdError.ExitError, fmt.Errorf("Test Failed: %+v", kubeclient.CoreV1().Pods(testNamespace).GetLogs("operator-test", &v1.PodLogOptions{})))
		}
	}
}
