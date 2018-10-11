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
	"time"

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

const (
	// ReconcilePeriodAnnotation - annotation used by a user to specify the reconcilation interval for the CR.
	// To use create a CR with an annotation "ansible.operator-sdk/reconcile-period: 30s" or some other valid
	// Duration. This will override the operators/or controllers reconcile period for that particular CR.
	ReconcilePeriodAnnotation = "ansible.operator-sdk/reconcile-period"
)

// AnsibleOperatorReconciler - object to reconcile runner requests
type AnsibleOperatorReconciler struct {
	GVK             schema.GroupVersionKind
	Runner          runner.Runner
	Client          client.Client
	EventHandlers   []events.EventHandler
	ReconcilePeriod time.Duration
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
	reconcileResult := reconcile.Result{RequeueAfter: r.ReconcilePeriod}
	if ds, ok := u.GetAnnotations()[ReconcilePeriodAnnotation]; ok {
		duration, err := time.ParseDuration(ds)
		if err != nil {
			return reconcileResult, err
		}
		reconcileResult.RequeueAfter = duration
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
		return reconcileResult, err
	}
	if !contains(pendingFinalizers, finalizer) && deleted {
		logrus.Info("Resource is terminated, skipping reconcilation")
		return reconcileResult, nil
	}

	spec := u.Object["spec"]
	_, ok := spec.(map[string]interface{})
	if !ok {
		logrus.Debugf("spec was not found")
		u.Object["spec"] = map[string]interface{}{}
		err = r.Client.Update(context.TODO(), u)
		if err != nil {
			return reconcileResult, err
		}
		reconcileResult.Requeue = true
		return reconcileResult, nil
	}
	status := u.Object["status"]
	_, ok = status.(map[string]interface{})
	if !ok {
		logrus.Debugf("status was not found")
		u.Object["status"] = map[string]interface{}{}
		err = r.Client.Update(context.TODO(), u)
		if err != nil {
			return reconcileResult, err
		}
		reconcileResult.Requeue = true
		return reconcileResult, nil
	}

	// If status is an empty map we can assume CR was just created
	if len(u.Object["status"].(map[string]interface{})) == 0 {
		logrus.Debugf("Setting phase status to %v", StatusPhaseCreating)
		u.Object["status"] = ResourceStatus{
			Phase: StatusPhaseCreating,
		}
		err = r.Client.Update(context.TODO(), u)
		if err != nil {
			return reconcileResult, err
		}
		reconcileResult.Requeue = true
		return reconcileResult, nil
	}

	ownerRef := metav1.OwnerReference{
		APIVersion: u.GetAPIVersion(),
		Kind:       u.GetKind(),
		Name:       u.GetName(),
		UID:        u.GetUID(),
	}

	kc, err := kubeconfig.Create(ownerRef, "http://localhost:8888", u.GetNamespace())
	if err != nil {
		return reconcileResult, err
	}
	defer os.Remove(kc.Name())
	eventChan, err := r.Runner.Run(u, kc.Name())
	if err != nil {
		return reconcileResult, err
	}

	// iterate events from ansible, looking for the final one
	statusEvent := eventapi.StatusJobEvent{}
	for event := range eventChan {
		for _, eHandler := range r.EventHandlers {
			go eHandler.Handle(u, event)
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
		return reconcileResult, err
	}

	// We only want to update the CustomResource once, so we'll track changes and do it at the end
	var needsUpdate bool
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
		needsUpdate = true
	}

	statusMap, ok := u.Object["status"].(map[string]interface{})
	if !ok {
		u.Object["status"] = ResourceStatus{
			Status: NewStatusFromStatusJobEvent(statusEvent),
		}
		logrus.Infof("adding status for the first time")
		needsUpdate = true
	} else {
		// Need to conver the map[string]interface into a resource status.
		if update, status := UpdateResourceStatus(statusMap, statusEvent); update {
			u.Object["status"] = status
			needsUpdate = true
		}
	}
	if needsUpdate {
		err = r.Client.Update(context.TODO(), u)
	}
	if !runSuccessful {
		reconcileResult.Requeue = true
		return reconcileResult, err
	}
	return reconcileResult, err
}

func contains(l []string, s string) bool {
	for _, elem := range l {
		if elem == s {
			return true
		}
	}
	return false
}
