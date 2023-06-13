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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	crHandler "sigs.k8s.io/controller-runtime/pkg/handler"
)

// LoggingEnqueueRequestForOwner wraps operator-lib handler for
// "InstrumentedEnqueueRequestForObject", and logs the events as they occur
//
//	&handler.LoggingEnqueueRequestForOwner{}
type LoggingEnqueueRequestForOwner struct {
	crHandler.EnqueueRequestForOwner
}

// Create implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForOwner) Create(c context.Context, e event.CreateEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Create", e.Object, nil)
	h.EnqueueRequestForOwner.Create(e, q)
}

// Update implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForOwner) Update(c context.Context, e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Update", e.ObjectOld, e.ObjectNew)
	h.EnqueueRequestForOwner.Update(e, q)
}

// Delete implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForOwner) Delete(c context.Context, e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Delete", e.Object, nil)
	h.EnqueueRequestForOwner.Delete(e, q)
}

// Generic implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForOwner) Generic(c context.Context, e event.GenericEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Generic", e.Object, nil)
	h.EnqueueRequestForOwner.Generic(e, q)
}

func (h LoggingEnqueueRequestForOwner) logEvent(eventType string, object, newObject client.Object) {
	ownerReference := extractTypedOwnerReference(h.EnqueueRequestForOwner.OwnerType.GetObjectKind().GroupVersionKind(), object.GetOwnerReferences())
	if ownerReference == nil && newObject != nil {
		ownerReference = extractTypedOwnerReference(h.EnqueueRequestForOwner.OwnerType.GetObjectKind().GroupVersionKind(), newObject.GetOwnerReferences())
	}

	// If no ownerReference was found then it's probably not an event we care about
	if ownerReference != nil {
		kvs := []interface{}{
			"Event type", eventType,
			"GroupVersionKind", object.GetObjectKind().GroupVersionKind().String(),
			"Name", object.GetName(),
		}
		if objectNs := object.GetNamespace(); objectNs != "" {
			kvs = append(kvs, "Namespace", objectNs)
		}
		kvs = append(kvs,
			"Owner APIVersion", ownerReference.APIVersion,
			"Owner Kind", ownerReference.Kind,
			"Owner Name", ownerReference.Name,
		)

		log.V(1).Info("OwnerReference handler event", kvs...)
	}
}

func extractTypedOwnerReference(ownerGVK schema.GroupVersionKind, ownerReferences []metav1.OwnerReference) *metav1.OwnerReference {
	for _, ownerRef := range ownerReferences {
		refGV, err := schema.ParseGroupVersion(ownerRef.APIVersion)
		if err != nil {
			log.Error(err, "Could not parse OwnerReference APIVersion",
				"api version", ownerRef.APIVersion)
		}

		if ownerGVK.Group == refGV.Group &&
			ownerGVK.Kind == ownerRef.Kind {
			return &ownerRef
		}
	}
	return nil
}
