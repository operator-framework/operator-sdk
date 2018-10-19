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
	"strings"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/pkg/test"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type testClusterConfig struct {
	namespace       string
	kubeconfig      string
	imagePullPolicy string
	serviceAccount  string
	pendingTimeout  int
}

var tcConfig testClusterConfig

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
	testCmd.Flags().StringVar(&tcConfig.namespace, "namespace", "default", "Namespace to run tests in")
	testCmd.Flags().StringVar(&tcConfig.kubeconfig, "kubeconfig", defaultKubeConfig, "Kubeconfig path")
	testCmd.Flags().StringVar(&tcConfig.imagePullPolicy, "image-pull-policy", "Always", "Set test pod image pull policy. Allowed values: Always, Never")
	testCmd.Flags().StringVar(&tcConfig.serviceAccount, "service-account", "default", "Service account to run tests on")
	testCmd.Flags().IntVar(&tcConfig.pendingTimeout, "pending-timeout", 60, "Timeout in seconds for testing pod to stay in pending state (default 60s)")

	return testCmd
}

func testClusterFunc(cmd *cobra.Command, args []string) error {
	// in main.go, we catch and print errors, so we don't want cobra to print the error itself
	cmd.SilenceErrors = true
	if len(args) != 1 {
		return fmt.Errorf("operator-sdk test cluster requires exactly 1 argument")
	}
	var pullPolicy v1.PullPolicy
	if strings.ToLower(tcConfig.imagePullPolicy) == "always" {
		pullPolicy = v1.PullAlways
	} else if strings.ToLower(tcConfig.imagePullPolicy) == "never" {
		pullPolicy = v1.PullNever
	} else {
		return fmt.Errorf("invalid imagePullPolicy '%v'", tcConfig.imagePullPolicy)
	}
	// cobra prints its help message on error; we silence that here because any errors below
	// are due to the test failing, not incorrect user input
	cmd.SilenceUsage = true
	testPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "operator-test",
		},
		Spec: v1.PodSpec{
			ServiceAccountName: tcConfig.serviceAccount,
			RestartPolicy:      v1.RestartPolicyNever,
			Containers: []v1.Container{{
				Name:            "operator-test",
				Image:           args[0],
				ImagePullPolicy: pullPolicy,
				Command:         []string{"/" + scaffold.GoTestScriptFile},
				Env: []v1.EnvVar{{
					Name:      test.TestNamespaceEnv,
					ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}},
				}},
			}},
		},
	}
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", tcConfig.kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to get kubeconfig: %v", err)
	}
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create kubeclient: %v", err)
	}
	testPod, err = kubeclient.CoreV1().Pods(tcConfig.namespace).Create(testPod)
	if err != nil {
		return fmt.Errorf("failed to create test pod: %v", err)
	}
	defer func() {
		err = kubeclient.CoreV1().Pods(tcConfig.namespace).Delete(testPod.Name, &metav1.DeleteOptions{})
		if err != nil {
			fmt.Printf("Warning: failed to delete test pod")
		}
	}()
	err = wait.Poll(time.Second*5, time.Second*time.Duration(tcConfig.pendingTimeout), func() (bool, error) {
		testPod, err = kubeclient.CoreV1().Pods(tcConfig.namespace).Get(testPod.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get test pod: %v", err)
		}
		if testPod.Status.Phase == v1.PodPending {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		testPod, err = kubeclient.CoreV1().Pods(tcConfig.namespace).Get(testPod.Name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get test pod: %v", err)
		}
		waitingState := testPod.Status.ContainerStatuses[0].State.Waiting
		return fmt.Errorf("test pod stuck in 'Pending' phase for longer than %d seconds.\nMessage: %s\nReason: %s", tcConfig.pendingTimeout, waitingState.Message, waitingState.Reason)
	}
	for {
		testPod, err = kubeclient.CoreV1().Pods(tcConfig.namespace).Get(testPod.Name, metav1.GetOptions{})
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
			req := kubeclient.CoreV1().Pods(tcConfig.namespace).GetLogs(testPod.Name, &v1.PodLogOptions{})
			readCloser, err := req.Stream()
			if err != nil {
				return fmt.Errorf("test failed and failed to get error logs")
			}
			defer readCloser.Close()
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(readCloser)
			if err != nil {
				return fmt.Errorf("test failed and failed to read pod logs: %v", err)
			}
			return fmt.Errorf("test failed:\n%s", buf.String())
		}
	}
}
