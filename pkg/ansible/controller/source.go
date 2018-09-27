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
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// ReconcileLoop - new loop
type ReconcileLoop struct {
	Source   chan event.GenericEvent
	Stop     <-chan struct{}
	GVK      schema.GroupVersionKind
	Interval time.Duration
	Client   client.Client
}

// NewReconcileLoop - loop for a GVK.
// The reconcilation loop is needed because the resync period
// for the informer is not suitable for this use case.
func NewReconcileLoop(interval time.Duration, gvk schema.GroupVersionKind, c client.Client) ReconcileLoop {
	s := make(chan event.GenericEvent, 1025)
	return ReconcileLoop{
		Source:   s,
		GVK:      gvk,
		Interval: interval,
		Client:   c,
	}
}

// Start - start the reconcile loop
func (r *ReconcileLoop) Start() {
	go func() {
		ticker := time.NewTicker(r.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// List all object for the GVK
				ul := &unstructured.UnstructuredList{}
				ul.SetGroupVersionKind(r.GVK)
				err := r.Client.List(context.Background(), nil, ul)
				if err != nil {
					logrus.Warningf("unable to list resources for GV: %v during reconcilation", r.GVK)
					continue
				}
				for _, u := range ul.Items {
					e := event.GenericEvent{
						Meta:   &u,
						Object: &u,
					}
					r.Source <- e
				}
			case <-r.Stop:
				return
			}
		}
	}()
}
