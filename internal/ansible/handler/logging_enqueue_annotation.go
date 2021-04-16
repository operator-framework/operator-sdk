package handler

import (
	"fmt"
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
//		&handler.LoggingEnqueueRequestForAnnotation{}
type LoggingEnqueueRequestForAnnotation struct {
	handler.EnqueueRequestForAnnotation
}

// Create implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForAnnotation) Create(e event.CreateEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Create", e.Object, nil)
	h.EnqueueRequestForAnnotation.Create(e, q)
}

// Update implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForAnnotation) Update(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Update", e.ObjectOld, e.ObjectNew)
	h.EnqueueRequestForAnnotation.Update(e, q)
}

// Delete implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForAnnotation) Delete(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Delete", e.Object, nil)
	h.EnqueueRequestForAnnotation.Delete(e, q)
}

// Generic implements EventHandler, and emits a log message.
func (h LoggingEnqueueRequestForAnnotation) Generic(e event.GenericEvent, q workqueue.RateLimitingInterface) {
	h.logEvent("Generic", e.Object, nil)
	h.EnqueueRequestForAnnotation.Generic(e, q)
}

func (h LoggingEnqueueRequestForAnnotation) logEvent(eventType string, object, newObject client.Object) {
	typeString, name, namespace := extractTypedOwnerAnnotations(h.EnqueueRequestForAnnotation.Type, object)
	if newObject != nil && typeString == "" {
		typeString, name, namespace = extractTypedOwnerAnnotations(h.EnqueueRequestForAnnotation.Type, newObject)
	}
	if namespace == "" {
		namespace = "<nil>"
	}

	objectNs := object.GetNamespace()
	if objectNs == "" {
		objectNs = "<nil>"
	}

	if name != "" && typeString != "" {
		log.Info(fmt.Sprintf("Received %s event for dependent resource with GVK %s with name %s in namespace %s, mapped to owner GK %s with name %s in namespace %s",
			eventType,
			object.GetObjectKind().GroupVersionKind().String(),
			object.GetName(),
			objectNs,
			typeString,
			name,
			namespace))
	}
}

func extractTypedOwnerAnnotations(ownerGK schema.GroupKind, object metav1.Object) (string, string, string) {
	if typeString, ok := object.GetAnnotations()[handler.TypeAnnotation]; ok && typeString == ownerGK.String() {
		if namespacedNameString, ok := object.GetAnnotations()[handler.NamespacedNameAnnotation]; ok {
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
