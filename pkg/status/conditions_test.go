// Copyright 2020 The Operator-SDK Authors
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

package status

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclock "k8s.io/apimachinery/pkg/util/clock"
)

var (
	initTime      time.Time
	clockInterval time.Duration
)

func init() {
	loc, _ := time.LoadLocation("Local")
	initTime = time.Date(2015, time.July, 11, 0, 1, 0, 0, loc)
	clockInterval = time.Hour
}

func initConditions(init ...Condition) Conditions {
	// Use the same initial time for all initial conditions
	clock = kubeclock.NewFakeClock(initTime)
	conditions := Conditions{}
	for _, c := range init {
		conditions.SetCondition(c)
	}

	// Use an incrementing clock for the rest of the test
	clock = &kubeclock.IntervalClock{
		Time:     initTime,
		Duration: clockInterval,
	}

	return conditions
}

func generateCondition(t ConditionType, s corev1.ConditionStatus) Condition {
	c := Condition{
		Type:    t,
		Status:  s,
		Reason:  ConditionReason(fmt.Sprintf("My%s%s", t, s)),
		Message: fmt.Sprintf("Condition %s is %s", t, s),
	}
	return c
}

func withLastTransitionTime(c Condition, t time.Time) Condition {
	c.LastTransitionTime = metav1.Time{Time: t}
	return c
}

func TestConditionDeepCopy(t *testing.T) {
	a := generateCondition("A", corev1.ConditionTrue)
	var aCopy Condition
	a.DeepCopyInto(&aCopy)
	if &a == &aCopy {
		t.Errorf("Expected and actual point to the same object: %p %#v", &a, &a)
	}
	if &a.Status == &aCopy.Status {
		t.Errorf("Expected and actual point to the same object: %p %#v", &a.Status, &a.Status)
	}
	if &a.Reason == &aCopy.Reason {
		t.Errorf("Expected and actual point to the same object: %p %#v", &a.Reason, &a.Reason)
	}
	if &a.Message == &aCopy.Message {
		t.Errorf("Expected and actual point to the same object: %p %#v", &a.Message, &a.Message)
	}
}

func TestConditionsSetEmpty(t *testing.T) {
	conditions := initConditions()

	setCondition := generateCondition("A", corev1.ConditionTrue)
	assert.True(t, conditions.SetCondition(setCondition))

	expectedCondition := withLastTransitionTime(setCondition, initTime.Add(clockInterval))
	actualCondition := conditions.GetCondition(setCondition.Type)
	assert.Equal(t, 1, len(conditions))
	assert.Equal(t, expectedCondition, *actualCondition)
}

func TestConditionsSetNotExists(t *testing.T) {
	conditions := initConditions(generateCondition("B", corev1.ConditionTrue))

	setCondition := generateCondition("A", corev1.ConditionTrue)
	assert.True(t, conditions.SetCondition(setCondition))

	expectedCondition := withLastTransitionTime(setCondition, initTime.Add(clockInterval))
	actualCondition := conditions.GetCondition(expectedCondition.Type)
	assert.Equal(t, 2, len(conditions))
	assert.Equal(t, expectedCondition, *actualCondition)
}

func TestConditionsSetExistsIdentical(t *testing.T) {
	existingCondition := generateCondition("A", corev1.ConditionTrue)
	conditions := initConditions(existingCondition)

	setCondition := existingCondition
	assert.False(t, conditions.SetCondition(setCondition))

	expectedCondition := withLastTransitionTime(setCondition, initTime)
	actualCondition := conditions.GetCondition(expectedCondition.Type)
	assert.Equal(t, 1, len(conditions))
	assert.Equal(t, expectedCondition, *actualCondition)
}
func TestConditionsSetExistsDifferentReason(t *testing.T) {
	existingCondition := generateCondition("A", corev1.ConditionTrue)
	conditions := initConditions(existingCondition)

	setCondition := existingCondition
	setCondition.Reason = "ChangedReason"
	assert.True(t, conditions.SetCondition(setCondition))

	expectedCondition := withLastTransitionTime(setCondition, initTime)
	actualCondition := conditions.GetCondition(expectedCondition.Type)
	assert.Equal(t, 1, len(conditions))
	assert.Equal(t, expectedCondition, *actualCondition)
}

func TestConditionsSetExistsDifferentStatus(t *testing.T) {
	existingCondition := generateCondition("A", corev1.ConditionTrue)
	conditions := initConditions(existingCondition)

	setCondition := existingCondition
	setCondition.Status = corev1.ConditionFalse
	setCondition.Reason = "ChangedReason"
	assert.True(t, conditions.SetCondition(setCondition))

	expectedCondition := withLastTransitionTime(setCondition, initTime.Add(clockInterval))
	actualCondition := conditions.GetCondition(expectedCondition.Type)
	assert.Equal(t, 1, len(conditions))
	assert.Equal(t, expectedCondition, *actualCondition)
}

func TestConditionsGetNotExists(t *testing.T) {
	conditions := initConditions(generateCondition("A", corev1.ConditionTrue))

	actualCondition := conditions.GetCondition(ConditionType("B"))
	assert.Nil(t, actualCondition)
}

func TestConditionsRemoveFromNilConditions(t *testing.T) {
	var conditions *Conditions = nil
	assert.False(t, conditions.RemoveCondition(ConditionType("C")))
}

func TestConditionsRemoveNotExists(t *testing.T) {
	conditions := initConditions(
		generateCondition("A", corev1.ConditionTrue),
		generateCondition("B", corev1.ConditionTrue),
	)

	assert.False(t, conditions.RemoveCondition(ConditionType("C")))
	a := conditions.GetCondition(ConditionType("A"))
	b := conditions.GetCondition(ConditionType("B"))
	assert.NotNil(t, a)
	assert.NotNil(t, b)
	assert.Equal(t, 2, len(conditions))
}

func TestConditionsRemoveExists(t *testing.T) {
	conditions := initConditions(
		generateCondition("A", corev1.ConditionTrue),
		generateCondition("B", corev1.ConditionTrue),
	)

	assert.True(t, conditions.RemoveCondition(ConditionType("A")))
	a := conditions.GetCondition(ConditionType("A"))
	b := conditions.GetCondition(ConditionType("B"))
	assert.Nil(t, a)
	assert.NotNil(t, b)
	assert.Equal(t, 1, len(conditions))
}

func TestConditionsIsTrueFor(t *testing.T) {
	conditions := NewConditions(
		generateCondition("False", corev1.ConditionFalse),
		generateCondition("True", corev1.ConditionTrue),
		generateCondition("Unknown", corev1.ConditionUnknown),
	)

	assert.True(t, conditions.IsTrueFor(ConditionType("True")))
	assert.False(t, conditions.IsTrueFor(ConditionType("False")))
	assert.False(t, conditions.IsTrueFor(ConditionType("Unknown")))
	assert.False(t, conditions.IsTrueFor(ConditionType("DoesNotExist")))
}

func TestConditionsIsFalseFor(t *testing.T) {
	conditions := NewConditions(
		generateCondition("False", corev1.ConditionFalse),
		generateCondition("True", corev1.ConditionTrue),
		generateCondition("Unknown", corev1.ConditionUnknown),
	)

	assert.False(t, conditions.IsFalseFor(ConditionType("True")))
	assert.True(t, conditions.IsFalseFor(ConditionType("False")))
	assert.False(t, conditions.IsFalseFor(ConditionType("Unknown")))
	assert.False(t, conditions.IsFalseFor(ConditionType("DoesNotExist")))
}

func TestConditionsIsUnknownFor(t *testing.T) {
	conditions := NewConditions(
		generateCondition("False", corev1.ConditionFalse),
		generateCondition("True", corev1.ConditionTrue),
		generateCondition("Unknown", corev1.ConditionUnknown),
	)

	assert.False(t, conditions.IsUnknownFor(ConditionType("True")))
	assert.False(t, conditions.IsUnknownFor(ConditionType("False")))
	assert.True(t, conditions.IsUnknownFor(ConditionType("Unknown")))
	assert.True(t, conditions.IsUnknownFor(ConditionType("DoesNotExist")))
}

func TestConditionsMarshalUnmarshalJSON(t *testing.T) {
	a := generateCondition("A", corev1.ConditionTrue)
	b := generateCondition("B", corev1.ConditionTrue)
	c := generateCondition("C", corev1.ConditionTrue)
	d := generateCondition("D", corev1.ConditionTrue)

	// Insert conditions unsorted
	conditions := initConditions(b, d, c, a)

	data, err := json.Marshal(conditions)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %s", err)
	}

	// Test that conditions are in sorted order by type.
	in := []Condition{}
	err = json.Unmarshal(data, &in)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %s", err)
	}
	assert.Equal(t, a.Type, in[0].Type)
	assert.Equal(t, b.Type, in[1].Type)
	assert.Equal(t, c.Type, in[2].Type)
	assert.Equal(t, d.Type, in[3].Type)

	// Test that the marshal/unmarshal cycle is lossless.
	unmarshalConds := Conditions{}
	err = json.Unmarshal(data, &unmarshalConds)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON: %s", err)
	}
	assert.Equal(t, conditions, unmarshalConds)
}
