// Copyright 2019 The Operator-SDK Authors
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

package predicate

import (
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var log = logf.Log.WithName("predicate")

var _ predicate.Predicate = DependentPredicate{}

// DependentPredicate is a predicate that filters events for resources
// created as dependents of a primary resource. It follows the following
// rules:
//
//   - Create events are ignored because it is assumed that the controller
//     reconciling the parent is the client creating the dependent
//     resources.
//   - Update events that change only the dependent resource status are
//     ignored because it is not typical for the controller of a primary
//     resource to write to the status of one its dependent resources.
//   - Deletion events are always handled because a controller will
//     typically want to recreate deleted dependent resources if the
//     primary resource is not deleted.
//   - Generic events are ignored.
//
// DependentPredicate is most often used in conjunction with
// controller-runtime's handler.EnqueueRequestForOwner
type DependentPredicate struct {
	predicate.Funcs
}

// Create filters out all events. It assumes that the controller
// reconciling the parent is the only client creating the dependent
// resources.
func (DependentPredicate) Create(e event.CreateEvent) bool {
	o := e.Object.(*unstructured.Unstructured)
	log.V(1).Info("Skipping reconciliation for dependent resource creation",
		"name", o.GetName(), "namespace", o.GetNamespace(), "apiVersion",
		o.GroupVersionKind().GroupVersion(), "kind", o.GroupVersionKind().Kind)
	return false
}

// Delete passes all events through. This allows the controller to
// recreate deleted dependent resources if the primary resource is
// not deleted.
func (DependentPredicate) Delete(e event.DeleteEvent) bool {
	o := e.Object.(*unstructured.Unstructured)
	log.V(1).Info("Reconciling due to dependent resource deletion",
		"name", o.GetName(), "namespace", o.GetNamespace(), "apiVersion",
		o.GroupVersionKind().GroupVersion(), "kind", o.GroupVersionKind().Kind)
	return true
}

// Generic filters out all events.
func (DependentPredicate) Generic(e event.GenericEvent) bool {
	o := e.Object.(*unstructured.Unstructured)
	log.V(1).Info("Skipping reconcile due to generic event", "name", o.GetName(),
		"namespace", o.GetNamespace(), "apiVersion", o.GroupVersionKind().GroupVersion(),
		"kind", o.GroupVersionKind().Kind)
	return false
}

// Update filters out events that change only the dependent resource
// status. It is not typical for the controller of a primary
// resource to write to the status of one its dependent resources.
func (DependentPredicate) Update(e event.UpdateEvent) bool {
	old := e.ObjectOld.(*unstructured.Unstructured).DeepCopy()
	new := e.ObjectNew.(*unstructured.Unstructured).DeepCopy()

	delete(old.Object, "status")
	delete(new.Object, "status")
	old.SetResourceVersion("")
	new.SetResourceVersion("")

	if reflect.DeepEqual(old.Object, new.Object) {
		return false
	}
	log.V(1).Info("Reconciling due to dependent resource update",
		"name", new.GetName(), "namespace", new.GetNamespace(), "apiVersion",
		new.GroupVersionKind().GroupVersion(), "kind", new.GroupVersionKind().Kind)
	return true
}
