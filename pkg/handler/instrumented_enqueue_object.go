// Copyright 2020 The Operator-SDK Authors
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

package handler

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	"github.com/operator-framework/operator-sdk/pkg/handler/internal/metrics"
)

// InstrumentedEnqueueRequestForObject wraps controller-runtime handler for "EnqueueRequestForObject", and
// sets up primary resource metrics on event handlers. The main objective of this handler is to set
// prometheues metrics when create/update/delete events occur. These metrics contain below information on resource.
//
// resource_created_at_seconds{"name", "namespace", "group", "version", "kind"}
//
// '&handler.InstrumentedEnqueueRequestForObject{}' is used to call the handler.
type InstrumentedEnqueueRequestForObject struct {
	handler.EnqueueRequestForObject
}

// Create implements EventHandler, and creates the metrics.
func (h InstrumentedEnqueueRequestForObject) Create(e event.CreateEvent, q workqueue.RateLimitingInterface) {
	setResourceMetric(e.Meta, e.Object)
	h.EnqueueRequestForObject.Create(e, q)
}

// Update implements EventHandler, and updates the metrics.
func (h InstrumentedEnqueueRequestForObject) Update(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	setResourceMetric(e.MetaOld, e.ObjectOld)
	setResourceMetric(e.MetaNew, e.ObjectNew)

	h.EnqueueRequestForObject.Update(e, q)
}

// Delete implements EventHandler, and deletes metrics.
func (h InstrumentedEnqueueRequestForObject) Delete(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	deleteResourceMetric(e.Meta, e.Object)
	h.EnqueueRequestForObject.Delete(e, q)
}

func setResourceMetric(metadata metav1.Object, obj runtime.Object) {
	if metadata != nil && obj != nil {
		labels := getResourceLabels(metadata, obj)
		m, _ := metrics.ResourceCreatedAt.GetMetricWith(labels)
		m.Set(float64(metadata.GetCreationTimestamp().UTC().Unix()))
	}
}

func deleteResourceMetric(metadata metav1.Object, obj runtime.Object) {
	if metadata != nil && obj != nil {
		labels := getResourceLabels(metadata, obj)
		_ = metrics.ResourceCreatedAt.Delete(labels)
	}
}

func getResourceLabels(metadata metav1.Object, obj runtime.Object) map[string]string {
	return map[string]string{
		"name":      metadata.GetName(),
		"namespace": metadata.GetNamespace(),
		"group":     obj.GetObjectKind().GroupVersionKind().Group,
		"version":   obj.GetObjectKind().GroupVersionKind().Version,
		"kind":      obj.GetObjectKind().GroupVersionKind().Kind,
	}
}
