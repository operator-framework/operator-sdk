// Copyright 2019 The Operator-SDK Authors
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

package kubemetrics

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

func newClientForGVK(cfg *rest.Config, apiVersion, kind string) (dynamic.NamespaceableResourceInterface, error) {
	apiResourceList, apiResource, err := getAPIResource(cfg, apiVersion, kind)
	if err != nil {
		return nil, fmt.Errorf("discovering resource information failed for %s in %s: %w", kind, apiVersion, err)
	}

	dc, err := newForConfig(cfg, apiResourceList.GroupVersion)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client failed for %s: %w", apiResourceList.GroupVersion, err)
	}

	gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
	if err != nil {
		return nil, fmt.Errorf("parsing GroupVersion %s failed: %w", apiResourceList.GroupVersion, err)
	}

	gvr := schema.GroupVersionResource{
		Group:    gv.Group,
		Version:  gv.Version,
		Resource: apiResource.Name,
	}

	return dc.Resource(gvr), nil
}

func getAPIResource(cfg *rest.Config, apiVersion, kind string) (*metav1.APIResourceList, *metav1.APIResource, error) {
	kclient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	_, apiResourceLists, err := kclient.Discovery().ServerGroupsAndResources()
	if err != nil {
		return nil, nil, err
	}

	for _, apiResourceList := range apiResourceLists {
		if apiResourceList.GroupVersion == apiVersion {
			for _, r := range apiResourceList.APIResources {
				if r.Kind == kind {
					return apiResourceList, &r, nil
				}
			}
		}
	}

	return nil, nil, fmt.Errorf("apiVersion %s and kind %s not found available in Kubernetes cluster",
		apiVersion, kind)
}

func newForConfig(c *rest.Config, groupVersion string) (dynamic.Interface, error) {
	config := rest.CopyConfig(c)

	err := setConfigDefaults(groupVersion, config)
	if err != nil {
		return nil, err
	}

	return dynamic.NewForConfig(config)
}

func setConfigDefaults(groupVersion string, config *rest.Config) error {
	gv, err := schema.ParseGroupVersion(groupVersion)
	if err != nil {
		return err
	}
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	if config.GroupVersion.Group == "" && config.GroupVersion.Version == "v1" {
		config.APIPath = "/api"
	}
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: scheme.Codecs}
	return nil
}
