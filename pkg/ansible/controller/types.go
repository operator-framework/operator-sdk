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

package controller

import (
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/eventapi"
)

const (
	host                = "localhost"
	StatusPhaseCreating = "Creating"
	StatusPhaseRunning  = "Running"
	StatusPhaseFailed   = "Failed"
)

type Status struct {
	Ok               int                `json:"ok"`
	Changed          int                `json:"changed"`
	Skipped          int                `json:"skipped"`
	Failures         int                `json:"failures"`
	TimeOfCompletion eventapi.EventTime `json:"completion"`
}

func NewStatusFromStatusJobEvent(je eventapi.StatusJobEvent) Status {
	// ok events.
	o := 0
	changed := 0
	skipped := 0
	failures := 0
	if v, ok := je.EventData.Changed[host]; ok {
		changed = v
	}
	if v, ok := je.EventData.Ok[host]; ok {
		o = v
	}
	if v, ok := je.EventData.Skipped[host]; ok {
		skipped = v
	}
	if v, ok := je.EventData.Failures[host]; ok {
		failures = v
	}
	return Status{
		Ok:               o,
		Changed:          changed,
		Skipped:          skipped,
		Failures:         failures,
		TimeOfCompletion: je.Created,
	}
}

func IsStatusEqual(s1, s2 Status) bool {
	return (s1.Ok == s2.Ok && s1.Changed == s2.Changed && s1.Skipped == s2.Skipped && s1.Failures == s2.Failures)
}

func NewStatusFromMap(sm map[string]interface{}) Status {
	//Create Old top level status
	// ok events.
	o := 0
	changed := 0
	skipped := 0
	failures := 0
	e := eventapi.EventTime{}
	if v, ok := sm["changed"]; ok {
		changed = int(v.(int64))
	}
	if v, ok := sm["ok"]; ok {
		o = int(v.(int64))
	}
	if v, ok := sm["skipped"]; ok {
		skipped = int(v.(int64))
	}
	if v, ok := sm["failures"]; ok {
		failures = int(v.(int64))
	}
	if v, ok := sm["completion"]; ok {
		s := v.(string)
		e.UnmarshalJSON([]byte(s))
	}
	return Status{
		Ok:               o,
		Changed:          changed,
		Skipped:          skipped,
		Failures:         failures,
		TimeOfCompletion: e,
	}
}

type ResourceStatus struct {
	Status         `json:",inline"`
	Phase          string   `json:"phase"`
	FailureMessage string   `json:"reason,omitempty"`
	History        []Status `json:"history,omitempty"`
}

func UpdateResourceStatus(sm map[string]interface{}, je eventapi.StatusJobEvent) (bool, ResourceStatus) {
	newStatus := NewStatusFromStatusJobEvent(je)
	oldStatus := NewStatusFromMap(sm)
	phase := StatusPhaseRunning
	// Don't update the status if new status and old status are equal.
	if IsStatusEqual(newStatus, oldStatus) {
		return false, ResourceStatus{}
	}

	history := []Status{}
	h, ok := sm["history"]
	if ok {
		hi := h.([]interface{})
		for _, m := range hi {
			ma := m.(map[string]interface{})
			history = append(history, NewStatusFromMap(ma))
		}
	}

	if newStatus.Failures > 0 {
		phase = StatusPhaseFailed
	}

	history = append(history, oldStatus)
	return true, ResourceStatus{
		Status:  newStatus,
		Phase:   phase,
		History: history,
	}
}
