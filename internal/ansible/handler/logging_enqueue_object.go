// Copyright 2021 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package handler

import (
	"context"

	"github.com/operator-framework/operator-lib/handler"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("ansible").WithName("handler")

// LoggingEnqueueRequestForObject wraps operator-lib handler for
// "InstrumentedEnqueueRequestForObject", and logs the events as they occur
//
//	&handler.LoggingEnqueueRequestForObject{}
type LoggingEnqueueRequestForObject struct {
	handler.InstrumentedEnqueueRequestForObject
}

// Create implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForObject) Create(ctx context.Context, e event.CreateEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Create", e.Object)
	h.InstrumentedEnqueueRequestForObject.Create(ctx, e, q)
}

// Update implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForObject) Update(ctx context.Context, e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Update", e.ObjectOld)
	h.InstrumentedEnqueueRequestForObject.Update(ctx, e, q)
}

// Delete implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForObject) Delete(ctx context.Context, e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Delete", e.Object)
	h.InstrumentedEnqueueRequestForObject.Delete(ctx, e, q)
}

// Generic implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForObject) Generic(ctx context.Context, e event.GenericEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Generic", e.Object)
	h.EnqueueRequestForObject.Generic(ctx, e, q)
}

func (h LoggingEnqueueRequestForObject) logEvent(eventType string, object client.Object) {
	kvs := []interface{}{
		"Event type", eventType,
		"GroupVersionKind", object.GetObjectKind().GroupVersionKind().String(),
		"Name", object.GetName(),
	}
	if objectNs := object.GetNamespace(); objectNs != "" {
		kvs = append(kvs, "Namespace", objectNs)
	}

	log.V(1).Info("Metrics handler event", kvs...)
}
