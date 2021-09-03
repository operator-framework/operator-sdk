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

package scorecard

import (
	"os"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	cruntime "sigs.k8s.io/controller-runtime/pkg/client/config"
)

// GetKubeClient will get a kubernetes client from the following sources:
// - a path to the kubeconfig file passed on the command line (--kubeconfig)
// - an environment variable that specifies the path (export KUBECONFIG)
// - the user's $HOME/.kube/config file
// - in-cluster connection for when the sdk is run within a cluster instead of
//   the command line
// TODO(joelanford): migrate scorecard use `internal/operator.Configuration`
func GetKubeClient(kubeconfig string) (client kubernetes.Interface, config *rest.Config, err error) {

	if kubeconfig != "" {
		os.Setenv(k8sutil.KubeConfigEnvVar, kubeconfig)
	}

	config, err = cruntime.GetConfig()
	if err != nil {
		return client, config, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return client, config, err
	}

	return clientset, config, err
}

// GetKubeNamespace returns the kubernetes namespace to use
// for scorecard pod creation
// the order of how the namespace is determined is as follows:
// - a namespace command line argument
// - a namespace determined from the kubeconfig file
// - the kubeconfig file is determined in the following order:
//   - from the kubeconfig flag if set
//   - from the KUBECONFIG env var if set
//   - from the $HOME/.kube/config path if exists
//   - returns 'default' as the namespace if not set in the kubeconfig
// TODO(joelanford): migrate scorecard to use `internal/operator.Configuration`
func GetKubeNamespace(kubeconfigPath, namespace string) string {

	if namespace != "" {
		return namespace
	}

	rules := clientcmd.NewDefaultClientConfigLoadingRules()

	if kubeconfigPath != "" {
		rules.ExplicitPath = kubeconfigPath
	}

	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})

	ns, _, err := kubeConfig.Namespace()
	if err != nil {
		return "default"
	}
	return ns

}
