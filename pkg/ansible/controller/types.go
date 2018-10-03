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
	"time"

	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/eventapi"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	host = "localhost"
)

// Phase - Phase for the CR managed by ansible operator.
type Phase string

const (
	// CreatingPhase - phase for ansible operator is creating the application.
	CreatingPhase Phase = "Creating"

	//FailingPhase -  phase for ansible operator has failed.
	FailingPhase Phase = "Failing"

	//DeletingPhase - phase for ansible operator status is deleting.
	DeletingPhase Phase = "Deleting"

	//RunningPhase - phase for ansible operator created the application and it is running.
	RunningPhase Phase = "Running"
)

// AnsibleStatus - ansible status
type AnsibleStatus struct {
	Ok               int                `json:"ok"`
	Changed          int                `json:"changed"`
	Skipped          int                `json:"skipped"`
	Failures         int                `json:"failures"`
	TimeOfCompletion eventapi.EventTime `json:"completion"`
}

// NewStatusFromStatusJobEvent - create new status from job event
func NewStatusFromStatusJobEvent(je eventapi.StatusJobEvent) *AnsibleStatus {
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
	return &AnsibleStatus{
		Ok:               o,
		Changed:          changed,
		Skipped:          skipped,
		Failures:         failures,
		TimeOfCompletion: je.Created,
	}
}

// newStatusFromMap - Create new status form map
func newStatusFromMap(sm map[string]interface{}) *AnsibleStatus {
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
	return &AnsibleStatus{
		Ok:               o,
		Changed:          changed,
		Skipped:          skipped,
		Failures:         failures,
		TimeOfCompletion: e,
	}
}

// Condition - Condition for the running application.
type Condition struct {
	AnsibleStatus      *AnsibleStatus   `json:"ansibleStatus,omitempty"`
	Phase              Phase            `json:"phase"`
	Messages           []FailureMessage `json:"messages,omitempty"`
	LastTransitionTime metav1.Time      `json:"lastTransitionTime"`
}

func newConditionFromMap(cm map[string]interface{}) Condition {
	asm, ok := cm["ansibleStatus"].(map[string]interface{})
	var as *AnsibleStatus
	if ok {
		as = newStatusFromMap(asm)
	}
	p := cm["phase"].(string)
	fmms, ok := cm["msgs"].([]interface{})
	var fms []FailureMessage
	if ok {
		for _, f := range fmms {
			fmm, _ := f.(map[string]interface{})
			fm := newFailureMessageFromMap(fmm)
			fms = append(fms, fm)
		}
	}
	ts := cm["lastTransitionTime"].(string)
	t := time.Time{}
	t.UnmarshalText([]byte(ts))
	return Condition{
		AnsibleStatus:      as,
		Phase:              Phase(p),
		Messages:           fms,
		LastTransitionTime: metav1.Time{Time: t},
	}
}

// ResourceStatus - Stautus for the CR managed by the operator.
type ResourceStatus struct {
	Conditions []Condition `json:"conditions"`
}

// FailureMessage - message for the failures that have occured.
type FailureMessage struct {
	TaskName  string      `json:"taskName"`
	Message   string      `json:"message"`
	Timestamp metav1.Time `json:"timestamp"`
}

func newFailureMessageFromMap(fm map[string]interface{}) FailureMessage {
	tn := fm["taskName"].(string)
	msg := fm["msg"].(string)
	ts := fm["timestamp"].(string)
	t := time.Time{}
	t.UnmarshalText([]byte(ts))
	return FailureMessage{
		TaskName:  tn,
		Message:   msg,
		Timestamp: metav1.Time{Time: t},
	}
}
