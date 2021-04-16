package handler

import (
	"fmt"

	"github.com/operator-framework/operator-lib/handler"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("ansible").WithName("handler")

// LoggingEnqueueRequestForObject wraps operator-lib handler for
// "InstrumentedEnqueueRequestForObject", and logs the events as they occur
//		&handler.LoggingEnqueueRequestForObject{}
type LoggingEnqueueRequestForObject struct {
	handler.InstrumentedEnqueueRequestForObject
}

// Create implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForObject) Create(e event.CreateEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Create", e.Object)
	h.InstrumentedEnqueueRequestForObject.Create(e, q)
}

// Update implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForObject) Update(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Update", e.ObjectOld)
	h.InstrumentedEnqueueRequestForObject.Update(e, q)
}

// Delete implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForObject) Delete(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Delete", e.Object)
	h.InstrumentedEnqueueRequestForObject.Delete(e, q)
}

// Generic implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForObject) Generic(e event.GenericEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Generic", e.Object)
	h.EnqueueRequestForObject.Generic(e, q)
}

func (h LoggingEnqueueRequestForObject) logEvent(eventType string, object client.Object) {
	objectNs := object.GetNamespace()
	if objectNs == "" {
		objectNs = "<nil>"
	}
	log.Info(fmt.Sprintf("Received %s event for GVK %s with name %s in namespace %s",
		eventType,
		object.GetObjectKind().GroupVersionKind().String(),
		object.GetName(), objectNs))

}
