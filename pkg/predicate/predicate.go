// Copyright 2018 The Operator-SDK Authors
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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/event"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var log = logf.Log.WithName("predicate").WithName("eventFilters")

// GenerationChangedPredicate implements a default update predicate function on generation change
// (adapted from sigs.k8s.io/controller-runtime/pkg/predicate/predicate.ResourceVersionChangedPredicate)
type GenerationChangedPredicate struct {
	predicate.Funcs
}

type ResourceFilterPredicate struct {
	predicate.Funcs
	Selector labels.Selector
}

// Update implements default UpdateEvent filter for validating generation change
func (GenerationChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.MetaOld == nil {
		log.Error(nil, "Update event has no old metadata", "event", e)
		return false
	}
	if e.ObjectOld == nil {
		log.Error(nil, "Update event has no old runtime object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		log.Error(nil, "Update event has no new runtime object for update", "event", e)
		return false
	}
	if e.MetaNew == nil {
		log.Error(nil, "Update event has no new metadata", "event", e)
		return false
	}
	if e.MetaNew.GetGeneration() == e.MetaOld.GetGeneration() && e.MetaNew.GetGeneration() != 0 {
		return false
	}
	return true
}

// Skips events that have labels matching selectors defined in watches.yaml
func (r ResourceFilterPredicate) eventFilter(eventLabels map[string]string) bool {
	return r.Selector.Matches(labels.Set(eventLabels))
}

func NewResourceFilterPredicate(s metav1.LabelSelector) (ResourceFilterPredicate, error) {
	selectorSpecs, err := metav1.LabelSelectorAsSelector(&s)
	requirements := ResourceFilterPredicate{Selector: selectorSpecs}
	return requirements, err

}

// Predicate functions that call the EventFilter Function
func (r ResourceFilterPredicate) Update(e event.UpdateEvent) bool {
	return r.eventFilter(e.MetaNew.GetLabels())
}

func (r ResourceFilterPredicate) Create(e event.CreateEvent) bool {
	return r.eventFilter(e.Meta.GetLabels())
}

func (r ResourceFilterPredicate) Delete(e event.DeleteEvent) bool {
	return r.eventFilter(e.Meta.GetLabels())
}

func (r ResourceFilterPredicate) Generic(e event.GenericEvent) bool {
	return r.eventFilter(e.Meta.GetLabels())
}
