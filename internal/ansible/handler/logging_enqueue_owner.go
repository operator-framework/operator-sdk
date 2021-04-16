package handler

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	crHandler "sigs.k8s.io/controller-runtime/pkg/handler"
)

// LoggingEnqueueRequestForOwner wraps operator-lib handler for
// "InstrumentedEnqueueRequestForObject", and logs the events as they occur
//		&handler.LoggingEnqueueRequestForOwner{}
type LoggingEnqueueRequestForOwner struct {
	crHandler.EnqueueRequestForOwner
}

// Create implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForOwner) Create(e event.CreateEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Create", e.Object, nil)
	h.EnqueueRequestForOwner.Create(e, q)
}

// Update implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForOwner) Update(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Update", e.ObjectOld, e.ObjectNew)
	h.EnqueueRequestForOwner.Update(e, q)
}

// Delete implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForOwner) Delete(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Delete", e.Object, nil)
	h.EnqueueRequestForOwner.Delete(e, q)
}

// Generic implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForOwner) Generic(e event.GenericEvent, q workqueue.RateLimitingInterface) {
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
		log.Info(fmt.Sprintf("Received %s event for dependent resource with GVK %s with name %s in namespace %s, mapped to owner GVK %s, Kind=%s with name %s",
			eventType,
			object.GetObjectKind().GroupVersionKind().String(),
			object.GetName(),
			object.GetNamespace(),
			ownerReference.APIVersion,
			ownerReference.Kind,
			ownerReference.Name))
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
