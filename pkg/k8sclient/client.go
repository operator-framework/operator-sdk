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

package k8sclient

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type resourceClientFactory struct {
	dynamicClient dynamic.Interface
	restMapper    *restmapper.DeferredDiscoveryRESTMapper
	kubeClient    kubernetes.Interface
	kubeConfig    *rest.Config
}

var (
	// this stores the singleton in a package local
	singletonFactory *resourceClientFactory
	once             sync.Once
)

// Private constructor for once.Do
func newSingletonFactory() {
	kubeClient, kubeConfig := mustNewKubeClientAndConfig()
	cachedDiscoveryClient := cached.NewMemCacheClient(kubeClient.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDiscoveryClient)
	restMapper.Reset()

	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	singletonFactory = &resourceClientFactory{
		kubeClient:    kubeClient,
		kubeConfig:    kubeConfig,
		dynamicClient: dynamicClient,
		restMapper:    restMapper,
	}
	singletonFactory.runBackgroundCacheReset(1 * time.Minute)
}

// GetResourceClient returns the resource client using a singleton factory
func GetResourceClient(apiVersion, kind, namespace string) (dynamic.ResourceInterface, string, error) {
	once.Do(newSingletonFactory)
	return singletonFactory.GetResourceClient(apiVersion, kind, namespace)
}

// GetKubeClient returns the kubernetes client used to create the dynamic client
func GetKubeClient() kubernetes.Interface {
	once.Do(newSingletonFactory)
	return singletonFactory.kubeClient
}

// GetKubeConfig returns the kubernetes rest configuration
func GetKubeConfig() *rest.Config {
	once.Do(newSingletonFactory)
	return singletonFactory.kubeConfig
}

// GetResourceClient returns the dynamic client and pluralName for the resource specified by the apiVersion and kind
func (c *resourceClientFactory) GetResourceClient(apiVersion, kind, namespace string) (dynamic.ResourceInterface, string, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse apiVersion: %v", err)
	}
	gvk := schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    kind,
	}

	gvr, err := gvkToGVR(gvk, c.restMapper)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get resource type: %v", err)
	}
	pluralName := gvr.Resource

	resourceClient := c.dynamicClient.Resource(*gvr).Namespace(namespace)
	return resourceClient, pluralName, nil
}

// apiResource consults the REST mapper to translate an <apiVersion, kind, namespace> tuple to a GroupVersionResource
func gvkToGVR(gvk schema.GroupVersionKind, restMapper *restmapper.DeferredDiscoveryRESTMapper) (*schema.GroupVersionResource, error) {
	mapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get the resource REST mapping for GroupVersionKind(%s): %v", gvk.String(), err)
	}
	return &mapping.Resource, nil
}

// mustNewKubeClientAndConfig returns the in-cluster config and kubernetes client
// or if KUBERNETES_CONFIG is given an out of cluster config and client
func mustNewKubeClientAndConfig() (kubernetes.Interface, *rest.Config) {
	var cfg *rest.Config
	var err error
	if os.Getenv(k8sutil.KubeConfigEnvVar) != "" {
		cfg, err = outOfClusterConfig()
	} else {
		cfg, err = inClusterConfig()
	}
	if err != nil {
		panic(err)
	}
	return kubernetes.NewForConfigOrDie(cfg), cfg
}

// inClusterConfig returns the in-cluster config accessible inside a pod
func inClusterConfig() (*rest.Config, error) {
	// Work around https://github.com/kubernetes/kubernetes/issues/40973
	// See https://github.com/coreos/etcd-operator/issues/731#issuecomment-283804819
	if len(os.Getenv("KUBERNETES_SERVICE_HOST")) == 0 {
		addrs, err := net.LookupHost("kubernetes.default.svc")
		if err != nil {
			return nil, err
		}
		os.Setenv("KUBERNETES_SERVICE_HOST", addrs[0])
	}
	if len(os.Getenv("KUBERNETES_SERVICE_PORT")) == 0 {
		os.Setenv("KUBERNETES_SERVICE_PORT", "443")
	}
	return rest.InClusterConfig()
}

func outOfClusterConfig() (*rest.Config, error) {
	kubeconfig := os.Getenv(k8sutil.KubeConfigEnvVar)
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	return config, err
}

// runBackgroundCacheReset - Starts the rest mapper cache reseting
// at a duration given.
func (c *resourceClientFactory) runBackgroundCacheReset(duration time.Duration) {
	ticker := time.NewTicker(duration)
	go func() {
		for range ticker.C {
			c.restMapper.Reset()
		}
	}()
}
