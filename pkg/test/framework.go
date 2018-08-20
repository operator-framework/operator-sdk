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
	goctx "context"
	"fmt"
	"sync"
	"time"

	extensions "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	extscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/kubernetes"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	dynclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// mutex for AddToFrameworkScheme
	mutex = sync.Mutex{}
	// Global framework struct
	Global *Framework
)

type Framework struct {
	KubeConfig        *rest.Config
	KubeClient        kubernetes.Interface
	ExtensionsClient  *extensions.Clientset
	Scheme            *runtime.Scheme
	RestMapper        *discovery.DeferredDiscoveryRESTMapper
	DynamicClient     dynclient.Client
	DynamicDecoder    runtime.Decoder
	NamespacedManPath *string
}

func setup(kubeconfigPath, namespacedManPath *string) error {
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
		KubeConfig:        kubeconfig,
		KubeClient:        kubeclient,
		ExtensionsClient:  extensionsClient,
		Scheme:            scheme,
		DynamicClient:     dynClient,
		DynamicDecoder:    dynDec,
		NamespacedManPath: namespacedManPath,
	}
	return nil
}

type addToSchemeFunc func(*runtime.Scheme) error

// AddToFrameworkScheme allows users to add the scheme for their custom resources
// to the framework's scheme for use with the dynamic client. The user provides
// the addToScheme function (located in the register.go file of their operator
// project) and the List struct for their custom resource. For example, for a
// memcached operator, the list stuct may look like:
// &MemcachedList{
//	TypeMeta: metav1.TypeMeta{
//		Kind: "Memcached",
//		APIVersion: "cache.example.com/v1alpha1",
//		},
//	}
// The List object is needed because the CRD has not always been fully registered
// by the time this function is called. If the CRD takes more than 5 seconds to
// become ready, this function throws an error
func AddToFrameworkScheme(addToScheme addToSchemeFunc, obj runtime.Object) error {
	mutex.Lock()
	defer mutex.Unlock()
	err := addToScheme(Global.Scheme)
	if err != nil {
		return err
	}
	cachedDiscoveryClient := cached.NewMemCacheClient(Global.KubeClient.Discovery())
	Global.RestMapper = discovery.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient, meta.InterfacesForUnstructured)
	Global.RestMapper.Reset()
	Global.DynamicClient, err = dynclient.New(Global.KubeConfig, dynclient.Options{Scheme: Global.Scheme, Mapper: Global.RestMapper})
	err = wait.PollImmediate(time.Second, time.Second*10, func() (done bool, err error) {
		err = Global.DynamicClient.List(goctx.TODO(), &dynclient.ListOptions{Namespace: "default"}, obj)
		if err != nil {
			Global.RestMapper.Reset()
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to build the dynamic client: %v", err)
	}
	Global.DynamicDecoder = serializer.NewCodecFactory(Global.Scheme).UniversalDeserializer()
	return nil
}
