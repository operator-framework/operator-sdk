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
	"os"
	"reflect"
	"strings"
	"time"

	libpredicate "github.com/operator-framework/operator-lib/predicate"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	ctrlpredicate "sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/operator-framework/operator-sdk/internal/ansible/events"
	"github.com/operator-framework/operator-sdk/internal/ansible/handler"
	"github.com/operator-framework/operator-sdk/internal/ansible/runner"
)

var log = logf.Log.WithName("ansible-controller")

// Options - options for your controller
type Options struct {
	EventHandlers               []events.EventHandler
	LoggingLevel                events.LogLevel
	Runner                      runner.Runner
	GVK                         schema.GroupVersionKind
	ReconcilePeriod             time.Duration
	ManageStatus                bool
	AnsibleDebugLogs            bool
	WatchDependentResources     bool
	WatchClusterScopedResources bool
	WatchAnnotationsChanges     bool
	MaxConcurrentReconciles     int
	Selector                    metav1.LabelSelector
}

// Add - Creates a new ansible operator controller and adds it to the manager
func Add(mgr manager.Manager, options Options) *controller.Controller {
	log.Info("Watching resource", "Options.Group", options.GVK.Group, "Options.Version",
		options.GVK.Version, "Options.Kind", options.GVK.Kind)
	if options.EventHandlers == nil {
		options.EventHandlers = []events.EventHandler{}
	}
	eventHandlers := append(options.EventHandlers, events.NewLoggingEventHandler(options.LoggingLevel))

	aor := &AnsibleOperatorReconciler{
		Client:                  mgr.GetClient(),
		GVK:                     options.GVK,
		Runner:                  options.Runner,
		EventHandlers:           eventHandlers,
		ReconcilePeriod:         options.ReconcilePeriod,
		ManageStatus:            options.ManageStatus,
		AnsibleDebugLogs:        options.AnsibleDebugLogs,
		APIReader:               mgr.GetAPIReader(),
		WatchAnnotationsChanges: options.WatchAnnotationsChanges,
	}

	scheme := mgr.GetScheme()
	_, err := scheme.New(options.GVK)
	if runtime.IsNotRegisteredError(err) {
		// Register the GVK with the schema
		scheme.AddKnownTypeWithName(options.GVK, &unstructured.Unstructured{})
		metav1.AddToGroupVersion(mgr.GetScheme(), schema.GroupVersion{
			Group:   options.GVK.Group,
			Version: options.GVK.Version,
		})
	} else if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	//Create new controller runtime controller and set the controller to watch GVK.
	c, err := controller.New(fmt.Sprintf("%v-controller", strings.ToLower(options.GVK.Kind)), mgr,
		controller.Options{
			Reconciler:              aor,
			MaxConcurrentReconciles: options.MaxConcurrentReconciles,
		})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Set up predicates.
	predicates := []ctrlpredicate.Predicate{
		ctrlpredicate.Or(ctrlpredicate.GenerationChangedPredicate{}, libpredicate.NoGenerationPredicate{}),
	}

	if options.WatchAnnotationsChanges {
		predicates = []ctrlpredicate.Predicate{
			ctrlpredicate.Or(ctrlpredicate.AnnotationChangedPredicate{}, predicates[0]),
		}
	}

	p, err := parsePredicateSelector(options.Selector)

	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if p != nil {
		predicates = append(predicates, p)
	}

	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(options.GVK)
	err = c.Watch(&source.Kind{Type: u}, &handler.LoggingEnqueueRequestForObject{}, predicates...)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	return &c
}

// parsePredicateSelector parses the selector in the WatchOptions and creates a predicate
// that is used to filter resources based on the specified selector
func parsePredicateSelector(selector metav1.LabelSelector) (ctrlpredicate.Predicate, error) {
	// If a selector has been specified in watches.yaml, add it to the watch's predicates.
	if !reflect.ValueOf(selector).IsZero() {
		p, err := ctrlpredicate.LabelSelectorPredicate(selector)
		if err != nil {
			return nil, fmt.Errorf("error constructing predicate from watches selector: %v", err)
		}
		return p, nil
	}
	return nil, nil
}
