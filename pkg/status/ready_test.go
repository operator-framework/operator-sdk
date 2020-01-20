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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestReadyCondition(t *testing.T) {
	assert.True(t, ReadyCondition(corev1.ConditionTrue).IsTrue())
	assert.True(t, ReadyCondition(corev1.ConditionFalse).IsFalse())
}

func TestConditionsIsReady(t *testing.T) {
	readyTrueConditions := initConditions(ReadyCondition(corev1.ConditionTrue))
	readyFalseConditions := initConditions(ReadyCondition(corev1.ConditionFalse))
	readyUnknownConditions := initConditions(ReadyCondition(corev1.ConditionUnknown))
	noReadyConditions := initConditions(Condition{Type: "Other", Status: corev1.ConditionTrue})

	assert.True(t, readyTrueConditions.IsReady())
	assert.False(t, readyFalseConditions.IsReady())
	assert.False(t, readyUnknownConditions.IsReady())
	assert.False(t, noReadyConditions.IsReady())
}
