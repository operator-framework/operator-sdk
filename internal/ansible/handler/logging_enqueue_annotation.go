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
	"strings"

	"github.com/operator-framework/operator-lib/handler"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// LoggingEnqueueRequestForAnnotation wraps operator-lib handler for
// "InstrumentedEnqueueRequestForObject", and logs the events as they occur
//
//	&handler.LoggingEnqueueRequestForAnnotation{}
type LoggingEnqueueRequestForAnnotation struct {
	handler.EnqueueRequestForAnnotation
}

// Create implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForAnnotation) Create(ctx context.Context, e event.CreateEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Create", e.Object, nil)
	h.EnqueueRequestForAnnotation.Create(ctx, e, q)
}

// Update implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForAnnotation) Update(ctx context.Context, e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Update", e.ObjectOld, e.ObjectNew)
	h.EnqueueRequestForAnnotation.Update(ctx, e, q)
}

// Delete implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForAnnotation) Delete(ctx context.Context, e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Delete", e.Object, nil)
	h.EnqueueRequestForAnnotation.Delete(ctx, e, q)
}

// Generic implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForAnnotation) Generic(ctx context.Context, e event.GenericEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Generic", e.Object, nil)
	h.EnqueueRequestForAnnotation.Generic(ctx, e, q)
}

func (h LoggingEnqueueRequestForAnnotation) logEvent(eventType string, object, newObject client.Object) {
	typeString, name, namespace := extractTypedOwnerAnnotations(h.EnqueueRequestForAnnotation.Type, object)
	if newObject != nil && typeString == "" {
		typeString, name, namespace = extractTypedOwnerAnnotations(h.EnqueueRequestForAnnotation.Type, newObject)
	}

	if name != "" && typeString != "" {
		kvs := []interface{}{
			"Event type", eventType,
			"GroupVersionKind", object.GetObjectKind().GroupVersionKind().String(),
			"Name", object.GetName(),
		}
		if objectNs := object.GetNamespace(); objectNs != "" {
			kvs = append(kvs, "Namespace", objectNs)
		}

		kvs = append(kvs,
			"Owner GroupKind", typeString,
			"Owner Name", name,
		)
		if namespace != "" {
			kvs = append(kvs, "Owner Namespace", namespace)
		}

		log.V(1).Info("Annotation handler event", kvs...)
	}
}

func extractTypedOwnerAnnotations(ownerGK schema.GroupKind, object metav1.Object) (string, string, string) {
	annotations := object.GetAnnotations()
	if len(annotations) == 0 {
		return "", "", ""
	}
	if typeString, ok := annotations[handler.TypeAnnotation]; ok && typeString == ownerGK.String() {
		if namespacedNameString, ok := annotations[handler.NamespacedNameAnnotation]; ok {
			parsed := parseNamespacedName(namespacedNameString)
			return typeString, parsed.Name, parsed.Namespace
		}
	}
	return "", "", ""
}

// parseNamespacedName parses the provided string to extract the namespace and name into a
// types.NamespacedName. The edge case of empty string is handled prior to calling this function.
func parseNamespacedName(namespacedNameString string) types.NamespacedName {
	values := strings.SplitN(namespacedNameString, "/", 2)

	switch len(values) {
	case 1:
		return types.NamespacedName{Name: values[0]}
	default:
		return types.NamespacedName{Namespace: values[0], Name: values[1]}
	}
}
