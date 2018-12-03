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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	crthandler "sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/operator-framework/operator-sdk/pkg/helm/release"
)

var log = logf.Log.WithName("helm.controller")

// WatchOptions contains the necessary values to create a new controller that
// manages helm releases in a particular namespace based on a GVK watch.
type WatchOptions struct {
	Namespace      string
	GVK            schema.GroupVersionKind
	ManagerFactory release.ManagerFactory
	ResyncPeriod   time.Duration
}

// Add creates a new helm operator controller and adds it to the manager
func Add(mgr manager.Manager, options WatchOptions) error {
	if options.ResyncPeriod == 0 {
		options.ResyncPeriod = time.Minute
	}
	r := &HelmOperatorReconciler{
		Client:         mgr.GetClient(),
		GVK:            options.GVK,
		ManagerFactory: options.ManagerFactory,
		ResyncPeriod:   options.ResyncPeriod,
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
	if err := c.Watch(&source.Kind{Type: o}, &crthandler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	log.Info("Watching resource", "apiVersion", options.GVK.GroupVersion(), "kind", options.GVK.Kind, "namespace", options.Namespace, "resyncPeriod", options.ResyncPeriod.String())
	return nil
}
