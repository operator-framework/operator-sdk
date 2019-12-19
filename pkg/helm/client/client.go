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

package client

import (
	"io"

	"helm.sh/helm/v3/pkg/kube"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	"k8s.io/client-go/discovery"
	cached "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// NewFromManager returns a Kubernetes client that can be used with
// a Tiller server.
func NewFromManager(mgr manager.Manager) (*kube.Client, error) {
	c, err := NewRESTClientGetter(mgr)
	if err != nil {
		return nil, err
	}
	return kube.New(c), nil
}

var _ genericclioptions.RESTClientGetter = &restClientGetter{}

type restClientGetter struct {
	restConfig      *rest.Config
	discoveryClient discovery.CachedDiscoveryInterface
	restMapper      meta.RESTMapper
}

func (c *restClientGetter) ToRESTConfig() (*rest.Config, error) {
	return c.restConfig, nil
}

func (c *restClientGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	return c.discoveryClient, nil
}

func (c *restClientGetter) ToRESTMapper() (meta.RESTMapper, error) {
	return c.restMapper, nil
}

func (c *restClientGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return nil
}

func NewRESTClientGetter(mgr manager.Manager) (genericclioptions.RESTClientGetter, error) {
	cfg := mgr.GetConfig()
	dc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}
	cdc := cached.NewMemCacheClient(dc)
	rm := mgr.GetRESTMapper()

	return &restClientGetter{
		restConfig:      cfg,
		discoveryClient: cdc,
		restMapper:      rm,
	}, nil
}

var _ kube.Interface = &ownerRefInjectingClient{}

func NewOwnerRefInjectingClient(base kube.Client, ownerRef metav1.OwnerReference) kube.Interface {
	return &ownerRefInjectingClient{
		refs:   []metav1.OwnerReference{ownerRef},
		Client: base,
	}
}

type ownerRefInjectingClient struct {
	refs []metav1.OwnerReference
	kube.Client
}

func (c *ownerRefInjectingClient) Build(reader io.Reader, validate bool) (kube.ResourceList, error) {
	resourceList, err := c.Client.Build(reader, validate)
	if err != nil {
		return resourceList, err
	}
	err = resourceList.Visit(func(r *resource.Info, err error) error {
		if err != nil {
			return err
		}
		objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(r.Object)
		if err != nil {
			return err
		}
		u := &unstructured.Unstructured{Object: objMap}
		if r.ResourceMapping().Scope == meta.RESTScopeNamespace {
			u.SetOwnerReferences(c.refs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resourceList, nil
}
