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

package controller_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/ansible/controller"
	ansiblestatus "github.com/operator-framework/operator-sdk/pkg/ansible/controller/status"
	"github.com/operator-framework/operator-sdk/pkg/ansible/events"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/eventapi"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/fake"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	gvk := schema.GroupVersionKind{
		Kind:    "Testing",
		Group:   "operator-sdk",
		Version: "v1beta1",
	}
	eventTime := time.Now()
	testCases := []struct {
		Name            string
		GVK             schema.GroupVersionKind
		ReconcilePeriod time.Duration
		ManageStatus    bool
		Runner          runner.Runner
		EventHandlers   []events.EventHandler
		Client          client.Client
		ExpectedObject  *unstructured.Unstructured
		Result          reconcile.Result
		Request         reconcile.Request
		ShouldError     bool
	}{
		{
			Name:            "cr not found",
			GVK:             gvk,
			ReconcilePeriod: 5 * time.Second,
			Runner: &fake.Runner{
				JobEvents: []eventapi.JobEvent{},
			},
			Client: fakeclient.NewFakeClient(),
			Result: reconcile.Result{},
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "not_found",
					Namespace: "default",
				},
			},
		},
		{
			Name:            "completed reconcile",
			GVK:             gvk,
			ReconcilePeriod: 5 * time.Second,
			ManageStatus:    true,
			Runner: &fake.Runner{
				JobEvents: []eventapi.JobEvent{
					eventapi.JobEvent{
						Event:   eventapi.EventPlaybookOnStats,
						Created: eventapi.EventTime{Time: eventTime},
					},
				},
			},
			Client: fakeclient.NewFakeClient(&unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "reconcile",
						"namespace": "default",
					},
					"apiVersion": "operator-sdk/v1beta1",
					"kind":       "Testing",
				},
			}),
			Result: reconcile.Result{
				RequeueAfter: 5 * time.Second,
			},
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "reconcile",
					Namespace: "default",
				},
			},
			ExpectedObject: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "reconcile",
						"namespace": "default",
					},
					"apiVersion": "operator-sdk/v1beta1",
					"kind":       "Testing",
					"spec":       map[string]interface{}{},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"status": "True",
								"type":   "Running",
								"ansibleResult": map[string]interface{}{
									"changed":    int64(0),
									"failures":   int64(0),
									"ok":         int64(0),
									"skipped":    int64(0),
									"completion": eventTime.Format("2006-01-02T15:04:05.99999999"),
								},
								"message": "Awaiting next reconciliation",
								"reason":  "Successful",
							},
						},
					},
				},
			},
		},
		{
			Name:            "Failure message reconcile",
			GVK:             gvk,
			ReconcilePeriod: 5 * time.Second,
			ManageStatus:    true,
			Runner: &fake.Runner{
				JobEvents: []eventapi.JobEvent{
					eventapi.JobEvent{
						Event:   eventapi.EventRunnerOnFailed,
						Created: eventapi.EventTime{Time: eventTime},
						EventData: map[string]interface{}{
							"res": map[string]interface{}{
								"msg": "new failure message",
							},
						},
					},
					eventapi.JobEvent{
						Event:   eventapi.EventPlaybookOnStats,
						Created: eventapi.EventTime{Time: eventTime},
					},
				},
			},
			Client: fakeclient.NewFakeClient(&unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "reconcile",
						"namespace": "default",
					},
					"apiVersion": "operator-sdk/v1beta1",
					"kind":       "Testing",
					"spec":       map[string]interface{}{},
				},
			}),
			Result: reconcile.Result{
				RequeueAfter: 5 * time.Second,
			},
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "reconcile",
					Namespace: "default",
				},
			},
			ExpectedObject: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "reconcile",
						"namespace": "default",
					},
					"apiVersion": "operator-sdk/v1beta1",
					"kind":       "Testing",
					"spec":       map[string]interface{}{},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"status": "True",
								"type":   "Failure",
								"ansibleResult": map[string]interface{}{
									"changed":    int64(0),
									"failures":   int64(0),
									"ok":         int64(0),
									"skipped":    int64(0),
									"completion": eventTime.Format("2006-01-02T15:04:05.99999999"),
								},
								"message": "new failure message",
								"reason":  "Failed",
							},
							map[string]interface{}{
								"status":  "False",
								"type":    "Running",
								"message": "Running reconciliation",
								"reason":  "Running",
							},
						},
					},
				},
			},
		},
		{
			Name:            "Finalizer successful reconcile",
			GVK:             gvk,
			ReconcilePeriod: 5 * time.Second,
			ManageStatus:    true,
			Runner: &fake.Runner{
				JobEvents: []eventapi.JobEvent{
					eventapi.JobEvent{
						Event:   eventapi.EventPlaybookOnStats,
						Created: eventapi.EventTime{Time: eventTime},
					},
				},
				Finalizer: "testing.io",
			},
			Client: fakeclient.NewFakeClient(&unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "reconcile",
						"namespace": "default",
						"annotations": map[string]interface{}{
							controller.ReconcilePeriodAnnotation: "3s",
						},
					},
					"apiVersion": "operator-sdk/v1beta1",
					"kind":       "Testing",
					"spec":       map[string]interface{}{},
				},
			}),
			Result: reconcile.Result{
				RequeueAfter: 3 * time.Second,
			},
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "reconcile",
					Namespace: "default",
				},
			},
			ExpectedObject: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "reconcile",
						"namespace": "default",
						"annotations": map[string]interface{}{
							controller.ReconcilePeriodAnnotation: "3s",
						},
						"finalizers": []interface{}{
							"testing.io",
						},
					},
					"apiVersion": "operator-sdk/v1beta1",
					"kind":       "Testing",
					"spec":       map[string]interface{}{},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"status": "True",
								"type":   "Running",
								"ansibleResult": map[string]interface{}{
									"changed":    int64(0),
									"failures":   int64(0),
									"ok":         int64(0),
									"skipped":    int64(0),
									"completion": eventTime.Format("2006-01-02T15:04:05.99999999"),
								},
								"message": "Awaiting next reconciliation",
								"reason":  "Successful",
							},
						},
					},
				},
			},
		},
		{
			Name:            "reconcile deletetion",
			GVK:             gvk,
			ReconcilePeriod: 5 * time.Second,
			Runner: &fake.Runner{
				JobEvents: []eventapi.JobEvent{
					eventapi.JobEvent{
						Event:   eventapi.EventPlaybookOnStats,
						Created: eventapi.EventTime{Time: eventTime},
					},
				},
				Finalizer: "testing.io",
			},
			Client: fakeclient.NewFakeClient(&unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "reconcile",
						"namespace": "default",
						"annotations": map[string]interface{}{
							controller.ReconcilePeriodAnnotation: "3s",
						},
						"deletionTimestamp": eventTime.Format(time.RFC3339),
					},
					"apiVersion": "operator-sdk/v1beta1",
					"kind":       "Testing",
					"spec":       map[string]interface{}{},
				},
			}),
			Result: reconcile.Result{},
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "reconcile",
					Namespace: "default",
				},
			},
		},
		{
			Name:            "Finalizer successful deletion reconcile",
			GVK:             gvk,
			ReconcilePeriod: 5 * time.Second,
			ManageStatus:    true,
			Runner: &fake.Runner{
				JobEvents: []eventapi.JobEvent{
					eventapi.JobEvent{
						Event:   eventapi.EventPlaybookOnStats,
						Created: eventapi.EventTime{Time: eventTime},
					},
				},
				Finalizer: "testing.io",
			},
			Client: fakeclient.NewFakeClient(&unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "reconcile",
						"namespace": "default",
						"finalizers": []interface{}{
							"testing.io",
						},
						"deletionTimestamp": eventTime.Format(time.RFC3339),
					},
					"apiVersion": "operator-sdk/v1beta1",
					"kind":       "Testing",
					"spec":       map[string]interface{}{},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"status": "True",
								"type":   "Running",
								"ansibleResult": map[string]interface{}{
									"changed":    int64(0),
									"failures":   int64(0),
									"ok":         int64(0),
									"skipped":    int64(0),
									"completion": eventTime.Format("2006-01-02T15:04:05.99999999"),
								},
								"message": "Awaiting next reconciliation",
								"reason":  "Successful",
							},
						},
					},
				},
			}),
			Result: reconcile.Result{
				RequeueAfter: 5 * time.Second,
			},
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "reconcile",
					Namespace: "default",
				},
			},
			ExpectedObject: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "reconcile",
						"namespace": "default",
					},
					"apiVersion": "operator-sdk/v1beta1",
					"kind":       "Testing",
					"spec":       map[string]interface{}{},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"status": "True",
								"type":   "Running",
								"ansibleResult": map[string]interface{}{
									"changed":    int64(0),
									"failures":   int64(0),
									"ok":         int64(0),
									"skipped":    int64(0),
									"completion": eventTime.Format("2006-01-02T15:04:05.99999999"),
								},
								"message": "Awaiting next reconciliation",
								"reason":  "Successful",
							},
						},
					},
				},
			},
		},
		{
			Name:            "No status event",
			GVK:             gvk,
			ReconcilePeriod: 5 * time.Second,
			Runner: &fake.Runner{
				JobEvents: []eventapi.JobEvent{
					eventapi.JobEvent{
						Created: eventapi.EventTime{Time: eventTime},
					},
				},
			},
			Client: fakeclient.NewFakeClient(&unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "reconcile",
						"namespace": "default",
					},
					"apiVersion": "operator-sdk/v1beta1",
					"kind":       "Testing",
					"spec":       map[string]interface{}{},
				},
			}),
			Result: reconcile.Result{
				RequeueAfter: 5 * time.Second,
			},
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "reconcile",
					Namespace: "default",
				},
			},
			ShouldError: true,
		},
		{
			Name:            "no manage status",
			GVK:             gvk,
			ReconcilePeriod: 5 * time.Second,
			ManageStatus:    false,
			Runner: &fake.Runner{
				JobEvents: []eventapi.JobEvent{
					eventapi.JobEvent{
						Event:   eventapi.EventPlaybookOnStats,
						Created: eventapi.EventTime{Time: eventTime},
					},
				},
			},
			Client: fakeclient.NewFakeClient(&unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "reconcile",
						"namespace": "default",
					},
					"apiVersion": "operator-sdk/v1beta1",
					"kind":       "Testing",
				},
			}),
			Result: reconcile.Result{
				RequeueAfter: 5 * time.Second,
			},
			Request: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "reconcile",
					Namespace: "default",
				},
			},
			ExpectedObject: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name":      "reconcile",
						"namespace": "default",
					},
					"apiVersion": "operator-sdk/v1beta1",
					"kind":       "Testing",
					"spec":       map[string]interface{}{},
					"status":     map[string]interface{}{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var aor reconcile.Reconciler = &controller.AnsibleOperatorReconciler{
				GVK:             tc.GVK,
				Runner:          tc.Runner,
				Client:          tc.Client,
				EventHandlers:   tc.EventHandlers,
				ReconcilePeriod: tc.ReconcilePeriod,
				ManageStatus:    tc.ManageStatus,
			}
			result, err := aor.Reconcile(tc.Request)
			if err != nil && !tc.ShouldError {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(result, tc.Result) {
				t.Fatalf("reconcile result does not equal\nexpected: %#v\nactual: %#v", tc.Result, result)
			}
			if tc.ExpectedObject != nil {
				actualObject := &unstructured.Unstructured{}
				actualObject.SetGroupVersionKind(tc.ExpectedObject.GroupVersionKind())
				tc.Client.Get(context.TODO(), types.NamespacedName{
					Name:      tc.ExpectedObject.GetName(),
					Namespace: tc.ExpectedObject.GetNamespace(),
				}, actualObject)
				if !reflect.DeepEqual(actualObject.GetAnnotations(), tc.ExpectedObject.GetAnnotations()) {
					t.Fatalf("annotations are not the same\nexpected: %v\nactual: %v", tc.ExpectedObject.GetAnnotations(), actualObject.GetAnnotations())
				}
				if !reflect.DeepEqual(actualObject.GetFinalizers(), tc.ExpectedObject.GetFinalizers()) &&
					len(actualObject.GetFinalizers()) != 0 && len(tc.ExpectedObject.GetFinalizers()) != 0 {
					t.Fatalf("finalizers are not the same\nexpected: %#v\nactual: %#v", tc.ExpectedObject.GetFinalizers(), actualObject.GetFinalizers())
				}
				sMap, _ := tc.ExpectedObject.Object["status"].(map[string]interface{})
				expectedStatus := ansiblestatus.CreateFromMap(sMap)
				sMap, _ = actualObject.Object["status"].(map[string]interface{})
				actualStatus := ansiblestatus.CreateFromMap(sMap)
				if len(expectedStatus.Conditions) != len(actualStatus.Conditions) {
					t.Fatalf("status conditions not the same\nexpected: %v\nactual: %v", expectedStatus, actualStatus)
				}
				for _, c := range expectedStatus.Conditions {
					actualCond := ansiblestatus.GetCondition(actualStatus, c.Type)
					if c.Reason != actualCond.Reason || c.Message != actualCond.Message || c.Status != actualCond.Status {
						t.Fatalf("message or reason did not match\nexpected: %v\nactual: %v", c, actualCond)
					}
					if c.AnsibleResult == nil && actualCond.AnsibleResult != nil {
						t.Fatalf("ansible result did not match expected: %v\nactual: %v", c.AnsibleResult, actualCond.AnsibleResult)
					}
					if c.AnsibleResult != nil {
						if !reflect.DeepEqual(c.AnsibleResult, actualCond.AnsibleResult) {
							t.Fatalf("ansible result did not match expected: %v\nactual: %v", c.AnsibleResult, actualCond.AnsibleResult)
						}
					}
				}
			}
		})
	}
}
