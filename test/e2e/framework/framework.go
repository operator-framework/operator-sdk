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

package framework

import (
	goctx "context"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

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
	KubeConfig     *rest.Config
	KubeClient     kubernetes.Interface
	Scheme         *runtime.Scheme
	DynamicClient  dynclient.Client
	DynamicDecoder runtime.Decoder
	ImageName      *string
}

func setup() error {
	defaultKubeConfig := ""
	homedir, ok := os.LookupEnv("HOME")
	if ok {
		defaultKubeConfig = homedir + "/.kube/config"
	}
	config := flag.String("kubeconfig", defaultKubeConfig, "kubeconfig path, defaults to $HOME/.kube/config")
	imageName := flag.String("image", "", "operator image name <repository>:<tag> used to push the image, defaults to none (builds image to local docker repo)")
	flag.Parse()
	if *config == "" {
		log.Fatalf("cannot find kubeconfig, exiting\n")
	}
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", *config)
	if err != nil {
		return err
	}
	kubeclient, err := kubernetes.NewForConfig(kubeconfig)
	if err != nil {
		return err
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
		KubeConfig:     kubeconfig,
		KubeClient:     kubeclient,
		Scheme:         scheme,
		DynamicClient:  dynClient,
		DynamicDecoder: dynDec,
		ImageName:      imageName,
	}
	return nil
}

type addToSchemeFunc func(*runtime.Scheme) error

func AddToFrameworkScheme(addToScheme addToSchemeFunc, obj runtime.Object) error {
	mutex.Lock()
	defer mutex.Unlock()
	err := addToScheme(Global.Scheme)
	if err != nil {
		return err
	}
	cachedDiscoveryClient := cached.NewMemCacheClient(Global.KubeClient.Discovery())
	restMapper := discovery.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient, meta.InterfacesForUnstructured)
	restMapper.Reset()
	Global.DynamicClient, err = dynclient.New(Global.KubeConfig, dynclient.Options{Scheme: Global.Scheme, Mapper: restMapper})
	err = wait.PollImmediate(time.Second, time.Second*10, func() (done bool, err error) {
		err = Global.DynamicClient.List(goctx.TODO(), &dynclient.ListOptions{Namespace: "default"}, obj)
		if err != nil {
			restMapper.Reset()
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
