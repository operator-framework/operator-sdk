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
	"strings"
	"time"

	ansiblestatus "github.com/operator-framework/operator-sdk/pkg/ansible/controller/status"
	"github.com/operator-framework/operator-sdk/pkg/ansible/events"
	"github.com/operator-framework/operator-sdk/pkg/ansible/proxy/kubeconfig"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner"
	"github.com/operator-framework/operator-sdk/pkg/ansible/runner/eventapi"

	"github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
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
		if err != nil {
			return reconcileResult, err
		}
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
	}
	statusInterface := u.Object["status"]
	statusMap, _ := statusInterface.(map[string]interface{})
	crStatus := ansiblestatus.CreateFromMap(statusMap)

	// If there is no current status add that we are working on this resource.
	errCond := ansiblestatus.GetCondition(crStatus, ansiblestatus.FailureConditionType)
	succCond := ansiblestatus.GetCondition(crStatus, ansiblestatus.RunningConditionType)

	// If the condition is currently running, making sure that the values are correct.
	// If they are the same a no-op, if they are different then it is a good thing we
	// are updating it.
	if (errCond == nil && succCond == nil) || (succCond != nil && succCond.Reason != ansiblestatus.SuccessfulReason) {
		c := ansiblestatus.NewCondition(
			ansiblestatus.RunningConditionType,
			v1.ConditionTrue,
			nil,
			ansiblestatus.RunningReason,
			ansiblestatus.RunningMessage,
		)
		ansiblestatus.SetCondition(&crStatus, *c)
		u.Object["status"] = crStatus
		err = r.Client.Update(context.TODO(), u)
		if err != nil {
			return reconcileResult, err
		}
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
	failureMessages := eventapi.FailureMessages{}
	for event := range eventChan {
		for _, eHandler := range r.EventHandlers {
			go eHandler.Handle(u, event)
		}
		if event.Event == eventapi.EventPlaybookOnStats {
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
		if event.Event == eventapi.EventRunnerOnFailed {
			failureMessages = append(failureMessages, event.GetFailedPlaybookMessage())
		}
	}
	if statusEvent.Event == "" {
		err := errors.New("did not receive playbook_on_stats event")
		logrus.Error(err.Error())
		return reconcileResult, err
	}

	// We only want to update the CustomResource once, so we'll track changes and do it at the end
	runSuccessful := len(failureMessages) == 0
	// The finalizer has run successfully, time to remove it
	if deleted && finalizerExists && runSuccessful {
		finalizers := []string{}
		for _, pendingFinalizer := range pendingFinalizers {
			if pendingFinalizer != finalizer {
				finalizers = append(finalizers, pendingFinalizer)
			}
		}
		u.SetFinalizers(finalizers)
		err := r.Client.Update(context.TODO(), u)
		if err != nil {
			return reconcileResult, err
		}
	}
	ansibleStatus := ansiblestatus.NewAnsibleResultFromStatusJobEvent(statusEvent)

	if !runSuccessful {
		sc := ansiblestatus.GetCondition(crStatus, ansiblestatus.RunningConditionType)
		sc.Status = v1.ConditionFalse
		ansiblestatus.SetCondition(&crStatus, *sc)
		c := ansiblestatus.NewCondition(
			ansiblestatus.FailureConditionType,
			v1.ConditionTrue,
			ansibleStatus,
			ansiblestatus.FailedReason,
			strings.Join(failureMessages, "\n"),
		)
		ansiblestatus.SetCondition(&crStatus, *c)
	} else {
		c := ansiblestatus.NewCondition(
			ansiblestatus.RunningConditionType,
			v1.ConditionTrue,
			ansibleStatus,
			ansiblestatus.SuccessfulReason,
			ansiblestatus.SuccessfulMessage,
		)
		// Remove the failure condition if set, because this completed successfully.
		ansiblestatus.RemoveCondition(&crStatus, ansiblestatus.FailureConditionType)
		ansiblestatus.SetCondition(&crStatus, *c)
	}
	// This needs the status subresource to be enabled by default.
	u.Object["status"] = crStatus
	err = r.Client.Update(context.TODO(), u)
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
