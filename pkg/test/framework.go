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

package test

import (
	"errors"
	"os"

	extensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var Global *Framework

type Framework struct {
	KubeConfig       *rest.Config
	KubeClient       kubernetes.Interface
	ExtensionsClient *extensions.Clientset
	Namespace        *string
	CrdManPath       *string
	OpManPath        *string
	RbacManPath      *string
}

func setup() error {
	kubeconfigEnv, ok := os.LookupEnv("TEST_KUBECONFIG")
	if !ok {
		return errors.New("Missing test environment variable; please run with `operator-sdk` test command")
	}
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeconfigEnv)
	if err != nil {
		return err
	}
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return err
	}
	extensionsClient, err := extensions.NewForConfig(kubeconfig)
	if err != nil {
		return err
	}
	namespace, ok := os.LookupEnv("TEST_NAMESPACE")
	if !ok {
		return errors.New("Missing test environment variable; please run with `operator-sdk` test command")
	}
	crdManPath, ok := os.LookupEnv("TEST_CRDMAN")
	if !ok {
		return errors.New("Missing test environment variable; please run with `operator-sdk` test command")
	}
	opManPath, ok := os.LookupEnv("TEST_OPMAN")
	if !ok {
		return errors.New("Missing test environment variable; please run with `operator-sdk` test command")
	}
	rbacManPath, ok := os.LookupEnv("TEST_RBACMAN")
	if !ok {
		return errors.New("Missing test environment variable; please run with `operator-sdk` test command")
	}
	Global = &Framework{
		KubeConfig:       kubeconfig,
		KubeClient:       kubeclient,
		ExtensionsClient: extensionsClient,
		Namespace:        &namespace,
		CrdManPath:       &crdManPath,
		OpManPath:        &opManPath,
		RbacManPath:      &rbacManPath,
	}
	return nil
}
