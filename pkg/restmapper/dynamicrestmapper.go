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

package restmapper

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

type DynamicRESTMapper struct {
	client   discovery.DiscoveryInterface
	delegate meta.RESTMapper
}

// NewDynamicRESTMapper returns a RESTMapper that dynamically discovers resource
// types at runtime. This is in contrast to controller-manager's default RESTMapper, which
// only checks resource types at startup, and so can't handle the case of first creating a
// CRD and then creating an instance of that CRD.
func NewDynamicRESTMapper(cfg *rest.Config) (meta.RESTMapper, error) {
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	drm := &DynamicRESTMapper{client: client}
	if err := drm.reload(); err != nil {
		return nil, err
	}
	return drm, nil
}

func (drm *DynamicRESTMapper) reload() error {
	gr, err := restmapper.GetAPIGroupResources(drm.client)
	if err != nil {
		return err
	}
	drm.delegate = restmapper.NewDiscoveryRESTMapper(gr)
	return nil
}

// reloadOnError checks if an error indicates that the delegated RESTMapper needs to be
// reloaded, and if so, reloads it and returns true.
func (drm *DynamicRESTMapper) reloadOnError(err error) bool {
	if _, matches := err.(*meta.NoKindMatchError); !matches {
		return false
	}
	err = drm.reload()
	if err != nil {
		utilruntime.HandleError(err)
	}
	return err == nil
}

func (drm *DynamicRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	gvk, err := drm.delegate.KindFor(resource)
	if drm.reloadOnError(err) {
		gvk, err = drm.delegate.KindFor(resource)
	}
	return gvk, err
}

func (drm *DynamicRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	gvks, err := drm.delegate.KindsFor(resource)
	if drm.reloadOnError(err) {
		gvks, err = drm.delegate.KindsFor(resource)
	}
	return gvks, err
}

func (drm *DynamicRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	gvr, err := drm.delegate.ResourceFor(input)
	if drm.reloadOnError(err) {
		gvr, err = drm.delegate.ResourceFor(input)
	}
	return gvr, err
}

func (drm *DynamicRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	gvrs, err := drm.delegate.ResourcesFor(input)
	if drm.reloadOnError(err) {
		gvrs, err = drm.delegate.ResourcesFor(input)
	}
	return gvrs, err
}

func (drm *DynamicRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	m, err := drm.delegate.RESTMapping(gk, versions...)
	if drm.reloadOnError(err) {
		m, err = drm.delegate.RESTMapping(gk, versions...)
	}
	return m, err
}

func (drm *DynamicRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	ms, err := drm.delegate.RESTMappings(gk, versions...)
	if drm.reloadOnError(err) {
		ms, err = drm.delegate.RESTMappings(gk, versions...)
	}
	return ms, err
}

func (drm *DynamicRESTMapper) ResourceSingularizer(resource string) (singular string, err error) {
	s, err := drm.delegate.ResourceSingularizer(resource)
	if drm.reloadOnError(err) {
		s, err = drm.delegate.ResourceSingularizer(resource)
	}
	return s, err
}
