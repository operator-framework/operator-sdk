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
	"context"
	"encoding/json"
	"errors"
	"os"
	"sort"

	"github.com/operator-framework/operator-sdk/pkg/ansible/events"
	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/kubeconfig"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/eventapi"

	"github.com/sirupsen/logrus"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// AnsibleOperatorReconciler - object to reconcile runner requests
type AnsibleOperatorReconciler struct {
	GVK           schema.GroupVersionKind
	Runner        runner.Runner
	Client        client.Client
	EventHandlers []events.EventHandler
}

// Reconcile - handle the event.
func (r *AnsibleOperatorReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(r.GVK)
	err := r.Client.Get(context.TODO(), request.NamespacedName, u)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}
	if err != nil {
		return reconcile.Result{}, err
	}

	deleted := u.GetDeletionTimestamp() != nil
	finalizer, finalizerExists := r.Runner.GetFinalizer()
	pendingFinalizers := u.GetFinalizers()
	// If the resource is being deleted we don't want to add the finalizer again
	if finalizerExists && !deleted && !contains(pendingFinalizers, finalizer) {
		logrus.Debugf("Adding finalizer %s to resource", finalizer)
		finalizers := append(pendingFinalizers, finalizer)
		u.SetFinalizers(finalizers)
		err := r.Client.Update(context.TODO(), u)
		return reconcile.Result{}, err
	}
	if !contains(pendingFinalizers, finalizer) && deleted {
		logrus.Info("Resource is terminated, skipping reconcilation")
		return reconcile.Result{}, nil
	}

	spec := u.Object["spec"]
	_, ok := spec.(map[string]interface{})
	if !ok {
		logrus.Debugf("spec was not found")
		u.Object["spec"] = map[string]interface{}{}
		err = r.Client.Update(context.TODO(), u)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// get the status and determine if we need the first update
	status, ok := getStatus(u)
	if ok {
		u.Object["status"] = status
		err = r.Client.Update(context.Background(), u)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// Set up kubeconfig for proxy.
	ownerRef := metav1.OwnerReference{
		APIVersion: u.GetAPIVersion(),
		Kind:       u.GetKind(),
		Name:       u.GetName(),
		UID:        u.GetUID(),
	}

	kc, err := kubeconfig.Create(ownerRef, "http://localhost:8888", u.GetNamespace())
	if err != nil {
		return reconcile.Result{}, err
	}
	defer os.Remove(kc.Name())
	eventChan, err := r.Runner.Run(u, kc.Name())
	if err != nil {
		return reconcile.Result{}, err
	}

	// iterate events from ansible, looking for the final one
	statusEvent := eventapi.StatusJobEvent{}
	messages := []FailureMessage{}
	for event := range eventChan {
		for _, eHandler := range r.EventHandlers {
			go eHandler.Handle(u, event)
		}
		if event.Event == "runner_on_failed" {
			// Need to pull out the res.msg from event data.
			result, ok := event.EventData["res"].(map[string]interface{})
			if !ok {
				logrus.Warningf("unable to find result for failure event")
				continue
			}
			t, _ := event.EventData["task"].(string)
			f := FailureMessage{
				TaskName:  t,
				Timestamp: metav1.Now(),
			}
			msg, ok := result["msg"].(string)
			f.Message = msg
			if !ok {
				logrus.Warningf("unable to find result for failure event")
				f.Message = "unknown error occured"
			}
			messages = append(messages, f)
		}
		if event.Event == "playbook_on_stats" {
			// convert to StatusJobEvent; would love a better way to do this
			data, err := json.Marshal(event)
			if err != nil {
				return reconcile.Result{}, err
			}
			err = json.Unmarshal(data, &statusEvent)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
	}
	if statusEvent.Event == "" {
		err := errors.New("did not receive playbook_on_stats event")
		logrus.Error(err.Error())
		return reconcile.Result{}, err
	}

	runSuccessful := true
	for _, count := range statusEvent.EventData.Failures {
		if count > 0 {
			runSuccessful = false
			break
		}
	}

	// The finalizer has run successfully, time to remove it
	if deleted && finalizerExists && runSuccessful {
		finalizers := []string{}
		for _, pendingFinalizer := range pendingFinalizers {
			if pendingFinalizer != finalizer {
				finalizers = append(finalizers, pendingFinalizer)
			}
		}
		u.SetFinalizers(finalizers)
		// Set status to deleting condition as well.
		status.Conditions = append(status.Conditions, Condition{
			LastTransitionTime: metav1.Now(),
			AnsibleStatus:      NewStatusFromStatusJobEvent(statusEvent),
			Phase:              DeletingPhase,
		})
		u.Object["status"] = status
		r.Client.Update(context.Background(), u)
		return reconcile.Result{}, nil
	}

	// If run was successful and the current condition is not successful then need to update condition.
	if runSuccessful && status.Conditions[len(status.Conditions)-1].Phase != RunningPhase {
		status.Conditions = append(status.Conditions, Condition{
			LastTransitionTime: metav1.Now(),
			AnsibleStatus:      NewStatusFromStatusJobEvent(statusEvent),
			Phase:              RunningPhase,
		})
		u.Object["status"] = status
		r.Client.Update(context.Background(), u)
		return reconcile.Result{}, err
	}

	// If run was not successful and current condition is not failed then need to update condition.
	if !runSuccessful && status.Conditions[len(status.Conditions)-1].Phase != FailingPhase {
		status.Conditions = append(status.Conditions, Condition{
			LastTransitionTime: metav1.Now(),
			AnsibleStatus:      NewStatusFromStatusJobEvent(statusEvent),
			Phase:              FailingPhase,
			Messages:           messages,
		})
		u.Object["status"] = status
		r.Client.Update(context.Background(), u)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, err
}

func contains(l []string, s string) bool {
	for _, elem := range l {
		if elem == s {
			return true
		}
	}
	return false
}

// getStatus - Retrieve the status object form the unstructured object
// Returns the status and if we need to update the status object for the first time.
func getStatus(u *unstructured.Unstructured) (ResourceStatus, bool) {
	// Add Status if not defined
	r := ResourceStatus{}
	s := u.Object["status"]
	sm, ok := s.(map[string]interface{})
	if !ok {
		logrus.Debugf("status was not found. Adding creating status.")
		r.Conditions = []Condition{
			Condition{
				Phase:              CreatingPhase,
				LastTransitionTime: metav1.Now(),
			},
		}
		return r, true
	}
	// Get the object out of the map.
	cms, ok := sm["conditions"].([]interface{})
	if !ok || len(cms) == 0 {
		logrus.Debugf("status was found. but did not have conditions..")
		r.Conditions = []Condition{
			Condition{
				Phase:              CreatingPhase,
				LastTransitionTime: metav1.Now(),
			},
		}
		return r, true
	}
	cs := []Condition{}
	for _, c := range cms {
		cm, _ := c.(map[string]interface{})
		c := newConditionFromMap(cm)
		cs = append(cs, c)
	}
	// Sort the slice
	sort.Slice(cs, func(i, j int) bool { return cs[i].LastTransitionTime.Before(&cs[j].LastTransitionTime) })
	r.Conditions = cs
	return r, false
}
