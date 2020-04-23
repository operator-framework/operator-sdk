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
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetKubeClient will get a kubernetes client from the following sources:
// - a path to the kubeconfig file passed on the command line (--kubeconfig)
// - an environment variable that specifies the path (export KUBECONFIG)
// - the user's $HOME/.kube/config file
// - in-cluster connection for when the sdk is run within a cluster instead of
//   the command line
func GetKubeClient(kubeconfig string) (client kubernetes.Interface, err error) {

	var inCluster bool

	if kubeconfig == "" {
		envVar := os.Getenv("KUBECONFIG")
		if envVar != "" {
			// use the KUBECONFIG env variable
			kubeconfig = envVar
		} else {

			home := homeDir()
			if home != "" {
				// use the $HOME/.kube/config path
				kubeconfig = filepath.Join(home, ".kube", "config")
			} else {
				// assume in-cluster
				inCluster = true
			}
		}
	}

	var config *rest.Config
	if inCluster {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
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
