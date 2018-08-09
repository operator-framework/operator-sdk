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
	"fmt"

	extensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var Global *Framework

type Framework struct {
	KubeConfig       *rest.Config
	KubeClient       kubernetes.Interface
	ExtensionsClient *extensions.Clientset
	DynamicClient    dynclient.Client
	DynamicDecoder   runtime.Decoder
	CrdManPath       *string
	OpManPath        *string
	RbacManPath      *string
}

func setup(kubeconfigPath, crdManPath, opManPath, rbacManPath *string) error {
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", *kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to build the kubeconfig: %v", err)
	}
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to build the kubeclient: %v", err)
	}
	extensionsClient, err := extensions.NewForConfig(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to build the extensionsClient: %v", err)
	}
	scheme := runtime.NewScheme()
	cgoscheme.AddToScheme(scheme)
	extscheme.AddToScheme(scheme)
	dynClient, err := dynclient.New(kubeconfig, dynclient.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("failed to build the dynamic client: %v", err)
	}
	dynDec := serializer.NewCodecFactory(scheme).UniversalDeserializer()
	Global = &Framework{
		KubeConfig:       kubeconfig,
		KubeClient:       kubeclient,
		ExtensionsClient: extensionsClient,
		DynamicClient:    dynClient,
		DynamicDecoder:   dynDec,
		CrdManPath:       crdManPath,
		OpManPath:        opManPath,
		RbacManPath:      rbacManPath,
	}
	return nil
}
