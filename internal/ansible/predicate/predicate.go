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
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type resourceFilterPredicate struct {
	predicate.Funcs
	Selector labels.Selector
}

// Skips events that have labels matching selectors defined in watches.yaml
func (r resourceFilterPredicate) eventFilter(eventLabels map[string]string) bool {
	return r.Selector.Matches(labels.Set(eventLabels))
}

func NewResourceFilterPredicate(s metav1.LabelSelector) (predicate.Predicate, error) {
	selectorSpecs, err := metav1.LabelSelectorAsSelector(&s)
	requirements := resourceFilterPredicate{Selector: selectorSpecs}
	return requirements, err

}

// Predicate functions that call the EventFilter Function
func (r resourceFilterPredicate) Update(e event.UpdateEvent) bool {
	return r.eventFilter(e.MetaNew.GetLabels())
}

func (r resourceFilterPredicate) Create(e event.CreateEvent) bool {
	return r.eventFilter(e.Meta.GetLabels())
}

func (r resourceFilterPredicate) Delete(e event.DeleteEvent) bool {
	return r.eventFilter(e.Meta.GetLabels())
}

func (r resourceFilterPredicate) Generic(e event.GenericEvent) bool {
	return r.eventFilter(e.Meta.GetLabels())
}
