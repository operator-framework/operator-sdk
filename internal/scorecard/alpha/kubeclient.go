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
	"os"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"k8s.io/client-go/kubernetes"
	cruntime "sigs.k8s.io/controller-runtime/pkg/client/config"
)

// GetKubeClient will get a kubernetes client from the following sources:
// - a path to the kubeconfig file passed on the command line (--kubeconfig)
// - an environment variable that specifies the path (export KUBECONFIG)
// - the user's $HOME/.kube/config file
// - in-cluster connection for when the sdk is run within a cluster instead of
//   the command line
func GetKubeClient(kubeconfig string) (client kubernetes.Interface, err error) {

	if kubeconfig != "" {
		os.Setenv(k8sutil.KubeConfigEnvVar, kubeconfig)
	}

	config, err := cruntime.GetConfig()
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
