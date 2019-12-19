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
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	yaml "gopkg.in/yaml.v2"
	rpb "helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	crthandler "sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/operator-framework/operator-sdk/pkg/helm/release"
	"github.com/operator-framework/operator-sdk/pkg/internal/predicates"
	"github.com/operator-framework/operator-sdk/pkg/predicate"
)

var log = logf.Log.WithName("helm.controller")

// WatchOptions contains the necessary values to create a new controller that
// manages helm releases in a particular namespace based on a GVK watch.
type WatchOptions struct {
	Namespace               string
	GVK                     schema.GroupVersionKind
	ManagerFactory          release.ManagerFactory
	ReconcilePeriod         time.Duration
	WatchDependentResources bool
}

// Add creates a new helm operator controller and adds it to the manager
func Add(mgr manager.Manager, options WatchOptions) error {
	r := &HelmOperatorReconciler{
		Client:          mgr.GetClient(),
		GVK:             options.GVK,
		ManagerFactory:  options.ManagerFactory,
		ReconcilePeriod: options.ReconcilePeriod,
	}

	// Register the GVK with the schema
	mgr.GetScheme().AddKnownTypeWithName(options.GVK, &unstructured.Unstructured{})
	metav1.AddToGroupVersion(mgr.GetScheme(), options.GVK.GroupVersion())

	controllerName := fmt.Sprintf("%v-controller", strings.ToLower(options.GVK.Kind))
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	o := &unstructured.Unstructured{}
	o.SetGroupVersionKind(options.GVK)
	if err := c.Watch(&source.Kind{Type: o}, &crthandler.EnqueueRequestForObject{}, predicate.GenerationChangedPredicate{}); err != nil {
		return err
	}

	if options.WatchDependentResources {
		watchDependentResources(mgr, r, c)
	}

	log.Info("Watching resource", "apiVersion", options.GVK.GroupVersion(), "kind", options.GVK.Kind, "namespace", options.Namespace, "reconcilePeriod", options.ReconcilePeriod.String())
	return nil
}

// watchDependentResources adds a release hook function to the HelmOperatorReconciler
// that adds watches for resources in released Helm charts.
func watchDependentResources(mgr manager.Manager, r *HelmOperatorReconciler, c controller.Controller) {
	owner := &unstructured.Unstructured{}
	owner.SetGroupVersionKind(r.GVK)

	// using predefined functions for filtering events
	dependentPredicate := predicates.DependentPredicateFuncs()

	var m sync.RWMutex
	watches := map[schema.GroupVersionKind]struct{}{}
	releaseHook := func(release *rpb.Release) error {
		dec := yaml.NewDecoder(bytes.NewBufferString(release.Manifest))
		for {
			var u unstructured.Unstructured
			err := dec.Decode(&u.Object)
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}

			gvk := u.GroupVersionKind()
			m.RLock()
			_, ok := watches[gvk]
			m.RUnlock()
			if ok {
				continue
			}

			restMapper := mgr.GetRESTMapper()
			depMapping, err := restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
			if err != nil {
				return err
			}
			ownerMapping, err := restMapper.RESTMapping(owner.GroupVersionKind().GroupKind(), owner.GroupVersionKind().Version)
			if err != nil {
				return err
			}

			depClusterScoped := depMapping.Scope.Name() == meta.RESTScopeNameRoot
			ownerClusterScoped := ownerMapping.Scope.Name() == meta.RESTScopeNameRoot

			if !ownerClusterScoped && depClusterScoped {
				m.Lock()
				watches[gvk] = struct{}{}
				m.Unlock()
				log.Info("Cannot watch cluster-scoped dependent resource for namespace-scoped owner. Changes to this dependent resource type will not be reconciled",
					"ownerApiVersion", r.GVK.GroupVersion(), "ownerKind", r.GVK.Kind, "apiVersion", gvk.GroupVersion(), "kind", gvk.Kind)
				continue
			}

			err = c.Watch(&source.Kind{Type: &u}, &crthandler.EnqueueRequestForOwner{OwnerType: owner}, dependentPredicate)
			if err != nil {
				return err
			}

			m.Lock()
			watches[gvk] = struct{}{}
			m.Unlock()
			log.Info("Watching dependent resource", "ownerApiVersion", r.GVK.GroupVersion(), "ownerKind", r.GVK.Kind, "apiVersion", gvk.GroupVersion(), "kind", gvk.Kind)
		}
	}
	r.releaseHook = releaseHook
}
