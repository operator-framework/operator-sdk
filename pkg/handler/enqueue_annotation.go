package handler

import (
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	crtHandler "sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// EnqueueRequestForAnnotation enqueues Requests based on the presence of an annotation that contains the
// namespaced name of the primary resource.
//
// The primary usecase for this, is to have a controller enqueue requests for the following scenarios
// 1. namespaced primary object and dependent cluster scoped resource
// 2. cluster scoped primary object.
// 3. namespaced primary object and dependent namespaced scoped but in a different namespace object.
type EnqueueRequestForAnnotation struct {
	// This is the annotation that will contain the namespaced/name or name if cluster scoped value for your resource.
	NamespaceNameAnnotation string

	mapper meta.RESTMapper
}

var _ crtHandler.EventHandler = &EnqueueRequestForAnnotation{}

// Create implements EventHandler
func (e *EnqueueRequestForAnnotation) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if ok, req := e.getAnnotationRequests(evt.Meta); ok {
		q.Add(req)
	}
}

// Update implements EventHandler
func (e *EnqueueRequestForAnnotation) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if ok, req := e.getAnnotationRequests(evt.MetaOld); ok {
		q.Add(req)
	}
	if ok, req := e.getAnnotationRequests(evt.MetaNew); ok {
		q.Add(req)
	}
}

// Delete implements EventHandler
func (e *EnqueueRequestForAnnotation) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if ok, req := e.getAnnotationRequests(evt.Meta); ok {
		q.Add(req)
	}
}

// Generic implements EventHandler
func (e *EnqueueRequestForAnnotation) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if ok, req := e.getAnnotationRequests(evt.Meta); ok {
		q.Add(req)
	}
}

func (e *EnqueueRequestForAnnotation) getAnnotationRequests(object metav1.Object) (bool, reconcile.Request) {
	if namespacedNameString, ok := object.GetAnnotations()[e.NamespaceNameAnnotation]; ok {
		if namespacedNameString == "" {
			return false, reconcile.Request{}
		}
		nsn := parseNamespacedName(namespacedNameString)
		return true, reconcile.Request{NamespacedName: nsn}
	}
	return false, reconcile.Request{}
}

func parseNamespacedName(namespacedNameString string) types.NamespacedName {
	values := strings.Split(namespacedNameString, "/")
	if len(values) == 1 {
		return types.NamespacedName{
			Name: values[0],
		}
	}
	if len(values) >= 2 {
		return types.NamespacedName{
			Name:      values[1],
			Namespace: values[0],
		}
	}
	return types.NamespacedName{}
}
