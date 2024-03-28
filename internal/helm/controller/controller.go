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

package controller

import (
	"fmt"
	"strings"
	"sync"
	"time"

	rpb "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/releaseutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	crthandler "sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/yaml"

	libhandler "github.com/operator-framework/operator-lib/handler"
	"github.com/operator-framework/operator-lib/predicate"
	"github.com/operator-framework/operator-sdk/internal/helm/release"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

var log = logf.Log.WithName("helm.controller")

// WatchOptions contains the necessary values to create a new controller that
// manages helm releases in a particular namespace based on a GVK watch.
type WatchOptions struct {
	GVK                     schema.GroupVersionKind
	ManagerFactory          release.ManagerFactory
	ReconcilePeriod         time.Duration
	WatchDependentResources bool
	OverrideValues          map[string]string
	SuppressOverrideValues  bool
	MaxConcurrentReconciles int
	Selector                metav1.LabelSelector
	DryRunOption            string
}

// Add creates a new helm operator controller and adds it to the manager
func Add(mgr manager.Manager, options WatchOptions) error {
	controllerName := fmt.Sprintf("%v-controller", strings.ToLower(options.GVK.Kind))

	r := &HelmOperatorReconciler{
		Client:                 mgr.GetClient(),
		EventRecorder:          mgr.GetEventRecorderFor(controllerName),
		GVK:                    options.GVK,
		ManagerFactory:         options.ManagerFactory,
		ReconcilePeriod:        options.ReconcilePeriod,
		OverrideValues:         options.OverrideValues,
		SuppressOverrideValues: options.SuppressOverrideValues,
		DryRunOption:           options.DryRunOption,
	}

	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: options.MaxConcurrentReconciles,
	})
	if err != nil {
		return err
	}

	o := &unstructured.Unstructured{}
	o.SetGroupVersionKind(options.GVK)

	if err := c.Watch(source.Kind(mgr.GetCache(), o), &libhandler.InstrumentedEnqueueRequestForObject{}); err != nil {
		return err
	}

	if options.WatchDependentResources {
		watchDependentResources(mgr, r, c)
	}

	log.Info("Watching resource", "apiVersion", options.GVK.GroupVersion(), "kind",
		options.GVK.Kind, "reconcilePeriod", options.ReconcilePeriod.String())
	return nil
}

// watchDependentResources adds a release hook function to the HelmOperatorReconciler
// that adds watches for resources in released Helm charts.
func watchDependentResources(mgr manager.Manager, r *HelmOperatorReconciler, c controller.Controller) {
	var m sync.RWMutex
	watches := map[schema.GroupVersionKind]struct{}{}
	releaseHook := func(release *rpb.Release) error {
		owner := &unstructured.Unstructured{}
		owner.SetGroupVersionKind(r.GVK)
		owner.SetNamespace(release.Namespace)
		resources := releaseutil.SplitManifests(release.Manifest)
		for _, resource := range resources {
			var u unstructured.Unstructured
			if err := yaml.Unmarshal([]byte(resource), &u); err != nil {
				return err
			}

			gvk := u.GroupVersionKind()
			if gvk.Empty() {
				continue
			}

			var setWatchOnResource = func(dependent runtime.Object) error {
				unstructuredObj := dependent.(*unstructured.Unstructured)
				gvkDependent := unstructuredObj.GroupVersionKind()
				if gvkDependent.Empty() {
					return nil
				}

				m.RLock()
				_, ok := watches[gvkDependent]
				m.RUnlock()
				if ok {
					return nil
				}

				restMapper := mgr.GetRESTMapper()
				useOwnerRef, err := k8sutil.SupportsOwnerReference(restMapper, owner, dependent, "")
				if err != nil {
					return err
				}

				if useOwnerRef { // Setup watch using owner references.
					err = c.Watch(source.Kind(mgr.GetCache(), unstructuredObj), crthandler.EnqueueRequestForOwner(mgr.GetScheme(), mgr.GetRESTMapper(), owner),
						predicate.DependentPredicate{})
					if err != nil {
						return err
					}
				} else { // Setup watch using annotations.
					err = c.Watch(source.Kind(mgr.GetCache(), unstructuredObj), &libhandler.EnqueueRequestForAnnotation{Type: gvkDependent.GroupKind()},
						predicate.DependentPredicate{})
					if err != nil {
						return err
					}
				}
				m.Lock()
				watches[gvkDependent] = struct{}{}
				m.Unlock()
				log.Info("Watching dependent resource", "ownerApiVersion", r.GVK.GroupVersion(),
					"ownerKind", r.GVK.Kind, "apiVersion", gvkDependent.GroupVersion(), "kind", gvkDependent.Kind)
				return nil
			}

			// List is not actually a resource and therefore cannot have a
			// watch on it. The watch will be on the kinds listed in the list
			// and will therefore need to be handled individually.
			listGVK := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "List"}
			if gvk == listGVK {
				errListItem := u.EachListItem(func(obj runtime.Object) error {
					return setWatchOnResource(obj)
				})
				if errListItem != nil {
					return errListItem
				}
			} else {
				err := setWatchOnResource(&u)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}
	r.releaseHook = releaseHook
}
