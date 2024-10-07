/*
Copyright 2020 The Operator-SDK Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package predicate

import (
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	crtpredicate "sigs.k8s.io/controller-runtime/pkg/predicate"
)

var log = logf.Log.WithName("predicate")

type GenerationChangedPredicate = crtpredicate.GenerationChangedPredicate

// DependentPredicateFuncs returns functions defined for filtering events
func DependentPredicateFuncs() crtpredicate.TypedFuncs[*unstructured.Unstructured] {
	dependentPredicate := crtpredicate.TypedFuncs[*unstructured.Unstructured]{
		// We don't need to reconcile dependent resource creation events
		// because dependent resources are only ever created during
		// reconciliation. Another reconcile would be redundant.
		CreateFunc: func(e event.TypedCreateEvent[*unstructured.Unstructured]) bool {
			o := e.Object
			log.V(1).Info("Skipping reconciliation for dependent resource creation", "name", o.GetName(), "namespace", o.GetNamespace(), "apiVersion", o.GroupVersionKind().GroupVersion(), "kind", o.GroupVersionKind().Kind)
			return false
		},

		// Reconcile when a dependent resource is deleted so that it can be
		// recreated.
		DeleteFunc: func(e event.TypedDeleteEvent[*unstructured.Unstructured]) bool {
			o := e.Object
			log.V(1).Info("Reconciling due to dependent resource deletion", "name", o.GetName(), "namespace", o.GetNamespace(), "apiVersion", o.GroupVersionKind().GroupVersion(), "kind", o.GroupVersionKind().Kind)
			return true
		},

		// Don't reconcile when a generic event is received for a dependent
		GenericFunc: func(e event.TypedGenericEvent[*unstructured.Unstructured]) bool {
			o := e.Object
			log.V(1).Info("Skipping reconcile due to generic event", "name", o.GetName(), "namespace", o.GetNamespace(), "apiVersion", o.GroupVersionKind().GroupVersion(), "kind", o.GroupVersionKind().Kind)
			return false
		},

		// Reconcile when a dependent resource is updated, so that it can
		// be patched back to the resource managed by the CR, if
		// necessary. Ignore updates that only change the status and
		// resourceVersion.
		UpdateFunc: func(e event.TypedUpdateEvent[*unstructured.Unstructured]) bool {
			old := e.ObjectOld.DeepCopy()
			updated := e.ObjectNew.DeepCopy()

			delete(old.Object, "status")
			delete(updated.Object, "status")
			old.SetResourceVersion("")
			updated.SetResourceVersion("")

			if reflect.DeepEqual(old.Object, updated.Object) {
				return false
			}
			log.V(1).Info("Reconciling due to dependent resource update", "name", updated.GetName(), "namespace", updated.GetNamespace(), "apiVersion", updated.GroupVersionKind().GroupVersion(), "kind", updated.GroupVersionKind().Kind)
			return true
		},
	}

	return dependentPredicate
}
