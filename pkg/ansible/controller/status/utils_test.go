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

package status

import (
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewCondition(t *testing.T) {
	testCases := []struct {
		name             string
		condType         ConditionType
		status           v1.ConditionStatus
		ansibleResult    *AnsibleResult
		reason           string
		message          string
		expectedCondtion Condition
	}{
		{
			name:          "running condition creating",
			condType:      RunningConditionType,
			status:        v1.ConditionTrue,
			ansibleResult: nil,
			reason:        RunningReason,
			message:       RunningMessage,
			expectedCondtion: Condition{
				Type:    RunningConditionType,
				Status:  v1.ConditionTrue,
				Reason:  RunningReason,
				Message: RunningMessage,
			},
		},
		{
			name:     "failure condition creating",
			condType: FailureConditionType,
			status:   v1.ConditionFalse,
			ansibleResult: &AnsibleResult{
				Changed:  0,
				Failures: 1,
				Ok:       10,
				Skipped:  1,
			},
			reason:  FailedReason,
			message: "invalid parameter",
			expectedCondtion: Condition{
				Type:    FailureConditionType,
				Status:  v1.ConditionFalse,
				Reason:  FailedReason,
				Message: "invalid parameter",
				AnsibleResult: &AnsibleResult{
					Changed:  0,
					Failures: 1,
					Ok:       10,
					Skipped:  1,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ac := NewCondition(tc.condType, tc.status, tc.ansibleResult, tc.reason, tc.message)
			tc.expectedCondtion.LastTransitionTime = ac.LastTransitionTime
			if !reflect.DeepEqual(*ac, tc.expectedCondtion) {
				t.Fatalf("condition did no match expected:\nActual: %#v\nExpected: %#v", *ac, tc.expectedCondtion)
			}
		})
	}
}

func TestGetCondition(t *testing.T) {
	testCases := []struct {
		name              string
		condType          ConditionType
		status            Status
		expectedCondition *Condition
	}{
		{
			name:     "find RunningCondition",
			condType: RunningConditionType,
			status: Status{
				Conditions: []Condition{
					Condition{
						Type: RunningConditionType,
					},
				},
			},
			expectedCondition: &Condition{
				Type: RunningConditionType,
			},
		},
		{
			name:     "did not find RunningCondition",
			condType: RunningConditionType,
			status: Status{
				Conditions: []Condition{
					Condition{
						Type: FailureConditionType,
					},
				},
			},
			expectedCondition: nil,
		},
		{
			name:     "find FailureCondition",
			condType: FailureConditionType,
			status: Status{
				Conditions: []Condition{
					Condition{
						Type: FailureConditionType,
					},
				},
			},
			expectedCondition: &Condition{
				Type: FailureConditionType,
			},
		},
		{
			name:     "did not find FailureCondition",
			condType: FailureConditionType,
			status: Status{
				Conditions: []Condition{
					Condition{
						Type: RunningConditionType,
					},
				},
			},
			expectedCondition: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ac := GetCondition(tc.status, tc.condType)
			if !reflect.DeepEqual(ac, tc.expectedCondition) {
				t.Fatalf("condition did no match expected:\nActual: %#v\nExpected: %#v", ac, tc.expectedCondition)
			}
		})
	}
}

func TestRemoveCondition(t *testing.T) {
	testCases := []struct {
		name         string
		condType     ConditionType
		status       Status
		expectedSize int
	}{
		{
			name:     "remove RunningCondition",
			condType: RunningConditionType,
			status: Status{
				Conditions: []Condition{
					Condition{
						Type: RunningConditionType,
					},
				},
			},
			expectedSize: 0,
		},
		{
			name:     "did not find RunningCondition",
			condType: RunningConditionType,
			status: Status{
				Conditions: []Condition{
					Condition{
						Type: FailureConditionType,
					},
				},
			},
			expectedSize: 1,
		},
		{
			name:     "remove FailureCondition",
			condType: FailureConditionType,
			status: Status{
				Conditions: []Condition{
					Condition{
						Type: FailureConditionType,
					},
				},
			},
			expectedSize: 0,
		},
		{
			name:     "did not find FailureCondition",
			condType: FailureConditionType,
			status: Status{
				Conditions: []Condition{
					Condition{
						Type: RunningConditionType,
					},
				},
			},
			expectedSize: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RemoveCondition(&tc.status, tc.condType)
			if tc.expectedSize != len(tc.status.Conditions) {
				t.Fatalf("conditions  did no match expected size:\nActual: %#v\nExpected: %#v", len(tc.status.Conditions), tc.expectedSize)
			}
		})
	}
}

func TestSetCondition(t *testing.T) {
	lastTransitionTime := metav1.Now()
	keeptMessage := SuccessfulMessage
	testCases := []struct {
		name                   string
		status                 *Status
		condition              *Condition
		expectedNewSize        int
		keepLastTransitionTime bool
		keepMessage            bool
	}{
		{
			name: "add new condition",
			status: &Status{
				Conditions: []Condition{},
			},
			condition:              NewCondition(RunningConditionType, v1.ConditionTrue, nil, RunningReason, RunningMessage),
			expectedNewSize:        1,
			keepLastTransitionTime: false,
		},
		{
			name: "update running condition",
			status: &Status{
				Conditions: []Condition{
					Condition{
						Type:               RunningConditionType,
						Status:             v1.ConditionTrue,
						Reason:             SuccessfulReason,
						Message:            SuccessfulMessage,
						LastTransitionTime: lastTransitionTime,
					},
				},
			},
			condition:              NewCondition(RunningConditionType, v1.ConditionTrue, nil, RunningReason, RunningMessage),
			expectedNewSize:        1,
			keepLastTransitionTime: true,
		},
		{
			name: "do not update running condition",
			status: &Status{
				Conditions: []Condition{
					Condition{
						Type:               RunningConditionType,
						Status:             v1.ConditionTrue,
						Reason:             RunningReason,
						Message:            SuccessfulMessage,
						LastTransitionTime: lastTransitionTime,
					},
				},
			},
			condition:              NewCondition(RunningConditionType, v1.ConditionTrue, nil, RunningReason, RunningMessage),
			expectedNewSize:        1,
			keepLastTransitionTime: true,
			keepMessage:            true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			SetCondition(tc.status, *tc.condition)
			if tc.expectedNewSize != len(tc.status.Conditions) {
				t.Fatalf("new size of conditions did not match expected\nActual: %v\nExpected: %v", len(tc.status.Conditions), tc.expectedNewSize)
			}
			if tc.keepLastTransitionTime {
				tc.condition.LastTransitionTime = lastTransitionTime
			}
			if tc.keepMessage {
				tc.condition.Message = keeptMessage
			}
			ac := GetCondition(*tc.status, tc.condition.Type)
			if !reflect.DeepEqual(ac, tc.condition) {
				t.Fatalf("condition did not match expected:\nActual: %#v\nExpected: %#v", ac, tc.condition)
			}
		})
	}
}
