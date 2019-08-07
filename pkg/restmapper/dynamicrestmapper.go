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
	"fmt"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// ErrRateLimited is returned by a DynamicRESTMapper method if the number
// of API calls has exceeded a limit within a certain time period.
type ErrRateLimited struct {
	// Duration to wait until the next API call can be made.
	Delay time.Duration
}

const errRLMsg = "too many API calls to the DynamicRESTMapper within a timeframe"

func (e ErrRateLimited) Error() string {
	return fmt.Sprintf("%s (%dns)", errRLMsg, int64(e.Delay))
}

var errRLRe = regexp.MustCompile(fmt.Sprintf(".*%s \\(([0-9]+)ns\\).*", errRLMsg))

func IsRateLimited(err error) (time.Duration, bool) {
	if e, ok := err.(ErrRateLimited); ok {
		return e.Delay, true
	}
	if matches := errRLRe.FindStringSubmatch(err.Error()); len(matches) > 1 {
		d, err := strconv.Atoi(matches[1])
		if err == nil {
			return time.Duration(d), true
		}
	}
	return 0, false
}

var (
	// LimitRate is the number of DynamicRESTMapper API calls allowed per second
	// assuming the rate of API calls <= LimitRate.
	// There is likely no need to change the default value.
	LimitRate = 600
	// LimitSize is the maximum number of simultaneous DynamicRESTMapper API
	// calls allowed.
	// There is likely no need to change the default value.
	LimitSize = 5
)

// DynamicRESTMapper is a RESTMapper that dynamically discovers resource
// types at runtime. This is in contrast to controller-manager's default
// RESTMapper, which only checks resource types at startup, and so can't
// handle the case of first creating a CRD and then creating an instance
// of that CRD.
type DynamicRESTMapper struct {
	client   discovery.DiscoveryInterface
	delegate meta.RESTMapper
	limiter  *limiter
}

// NewDynamicRESTMapper returns a DynamicRESTMapper for cfg.
func NewDynamicRESTMapper(cfg *rest.Config) (meta.RESTMapper, error) {
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, err
	}

	drm := &DynamicRESTMapper{
		client: client,
		limiter: &limiter{
			rate.NewLimiter(rate.Limit(LimitRate), LimitSize),
		},
	}
	if err := drm.setDelegate(); err != nil {
		return nil, err
	}
	return drm, nil
}

func (drm *DynamicRESTMapper) setDelegate() error {
	gr, err := restmapper.GetAPIGroupResources(drm.client)
	if err != nil {
		return err
	}
	drm.delegate = restmapper.NewDiscoveryRESTMapper(gr)
	return nil
}

func noKindMatchError(err error) bool {
	_, ok := err.(*meta.NoKindMatchError)
	return ok
}

// reload reloads the delegated RESTMapper, and will return an error only
// if a rate limit has been hit.
func (drm *DynamicRESTMapper) reload() error {
	if err := drm.limiter.checkRate(); err != nil {
		return err
	}
	if err := drm.setDelegate(); err != nil {
		utilruntime.HandleError(err)
	}
	return nil
}

func (drm *DynamicRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	gvk, err := drm.delegate.KindFor(resource)
	if noKindMatchError(err) {
		if rerr := drm.reload(); rerr != nil {
			return schema.GroupVersionKind{}, rerr
		}
		gvk, err = drm.delegate.KindFor(resource)
	}
	return gvk, err
}

func (drm *DynamicRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	gvks, err := drm.delegate.KindsFor(resource)
	if noKindMatchError(err) {
		if rerr := drm.reload(); rerr != nil {
			return nil, rerr
		}
		gvks, err = drm.delegate.KindsFor(resource)
	}
	return gvks, err
}

func (drm *DynamicRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	gvr, err := drm.delegate.ResourceFor(input)
	if noKindMatchError(err) {
		if rerr := drm.reload(); rerr != nil {
			return schema.GroupVersionResource{}, rerr
		}
		gvr, err = drm.delegate.ResourceFor(input)
	}
	return gvr, err
}

func (drm *DynamicRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	gvrs, err := drm.delegate.ResourcesFor(input)
	if noKindMatchError(err) {
		if rerr := drm.reload(); rerr != nil {
			return nil, rerr
		}
		gvrs, err = drm.delegate.ResourcesFor(input)
	}
	return gvrs, err
}

func (drm *DynamicRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	m, err := drm.delegate.RESTMapping(gk, versions...)
	if noKindMatchError(err) {
		if rerr := drm.reload(); rerr != nil {
			return nil, rerr
		}
		m, err = drm.delegate.RESTMapping(gk, versions...)
	}
	return m, err
}

func (drm *DynamicRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	ms, err := drm.delegate.RESTMappings(gk, versions...)
	if noKindMatchError(err) {
		if rerr := drm.reload(); rerr != nil {
			return nil, rerr
		}
		ms, err = drm.delegate.RESTMappings(gk, versions...)
	}
	return ms, err
}

func (drm *DynamicRESTMapper) ResourceSingularizer(resource string) (singular string, err error) {
	s, err := drm.delegate.ResourceSingularizer(resource)
	if noKindMatchError(err) {
		if rerr := drm.reload(); rerr != nil {
			return "", rerr
		}
		s, err = drm.delegate.ResourceSingularizer(resource)
	}
	return s, err
}

type limiter struct {
	*rate.Limiter
}

func (b *limiter) checkRate() error {
	res := b.Reserve()
	if res.Delay() == 0 {
		return nil
	}
	return ErrRateLimited{res.Delay()}
}
