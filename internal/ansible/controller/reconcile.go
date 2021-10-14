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
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	ansiblestatus "github.com/operator-framework/operator-sdk/internal/ansible/controller/status"
	"github.com/operator-framework/operator-sdk/internal/ansible/events"
	"github.com/operator-framework/operator-sdk/internal/ansible/metrics"
	"github.com/operator-framework/operator-sdk/internal/ansible/proxy/kubeconfig"
	"github.com/operator-framework/operator-sdk/internal/ansible/runner"
	"github.com/operator-framework/operator-sdk/internal/ansible/runner/eventapi"
)

const (
	// ReconcilePeriodAnnotation - annotation used by a user to specify the reconciliation interval for the CR.
	// To use create a CR with an annotation "ansible.sdk.operatorframework.io/reconcile-period: 30s" or some other valid
	// Duration. This will override the operators/or controllers reconcile period for that particular CR.
	ReconcilePeriodAnnotation = "ansible.sdk.operatorframework.io/reconcile-period"
)

// AnsibleOperatorReconciler - object to reconcile runner requests
type AnsibleOperatorReconciler struct {
	GVK              schema.GroupVersionKind
	Runner           runner.Runner
	Client           client.Client
	APIReader        client.Reader
	EventHandlers    []events.EventHandler
	ReconcilePeriod  time.Duration
	ManageStatus     bool
	AnsibleDebugLogs bool
}

// Reconcile - handle the event.
func (r *AnsibleOperatorReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) { //nolint:gocyclo
	// TODO: Try to reduce the complexity of this last measured at 42 (failing at > 30) and remove the // nolint:gocyclo
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(r.GVK)
	err := r.Client.Get(ctx, request.NamespacedName, u)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}
	if err != nil {
		return reconcile.Result{}, err
	}

	ident := strconv.Itoa(rand.Int())
	logger := logf.Log.WithName("reconciler").WithValues(
		"job", ident,
		"name", u.GetName(),
		"namespace", u.GetNamespace(),
	)

	reconcileResult := reconcile.Result{RequeueAfter: r.ReconcilePeriod}
	if ds, ok := u.GetAnnotations()[ReconcilePeriodAnnotation]; ok {
		duration, err := time.ParseDuration(ds)
		if err != nil {
			// Should attempt to update to a failed condition
			errmark := r.markError(ctx, request.NamespacedName, u,
				fmt.Sprintf("Unable to parse reconcile period annotation: %v", err))
			if errmark != nil {
				logger.Error(errmark, "Unable to mark error annotation")
			}
			logger.Error(err, "Unable to parse reconcile period annotation")
			return reconcileResult, err
		}
		reconcileResult.RequeueAfter = duration
	}

	deleted := u.GetDeletionTimestamp() != nil
	finalizer, finalizerExists := r.Runner.GetFinalizer()
	if !controllerutil.ContainsFinalizer(u, finalizer) {
		if deleted {
			// If the resource is being deleted we don't want to add the finalizer again
			logger.Info("Resource is terminated, skipping reconciliation")
			return reconcile.Result{}, nil
		} else if finalizerExists {
			logger.V(1).Info("Adding finalizer to resource", "Finalizer", finalizer)
			controllerutil.AddFinalizer(u, finalizer)
			err := r.Client.Update(ctx, u)
			if err != nil {
				logger.Error(err, "Unable to update cr with finalizer")
				return reconcileResult, err
			}
		}
	}

	spec := u.Object["spec"]
	_, ok := spec.(map[string]interface{})
	// Need to handle cases where there is no spec.
	// We can add the spec to the object, which will allow
	// everything to work, and will not get updated.
	// Therefore we can now deal with the case of secrets and configmaps.
	if !ok {
		logger.V(1).Info("Spec was not found")
		u.Object["spec"] = map[string]interface{}{}
	}

	if r.ManageStatus {
		errmark := r.markRunning(ctx, request.NamespacedName, u)
		if errmark != nil {
			logger.Error(errmark, "Unable to update the status to mark cr as running")
			return reconcileResult, errmark
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
		errmark := r.markError(ctx, request.NamespacedName, u, "Unable to run reconciliation")
		if errmark != nil {
			logger.Error(errmark, "Unable to mark error to run reconciliation")
		}
		logger.Error(err, "Unable to generate kubeconfig")
		return reconcileResult, err
	}
	defer func() {
		if err := os.Remove(kc.Name()); err != nil {
			logger.Error(err, "Failed to remove generated kubeconfig file")
		}
	}()
	result, err := r.Runner.Run(ident, u, kc.Name())
	if err != nil {
		errmark := r.markError(ctx, request.NamespacedName, u, "Unable to run reconciliation")
		if errmark != nil {
			logger.Error(errmark, "Unable to mark error to run reconciliation")
		}
		logger.Error(err, "Unable to run ansible runner")
		return reconcileResult, err
	}

	// iterate events from ansible, looking for the final one
	statusEvent := eventapi.StatusJobEvent{}
	failureMessages := eventapi.FailureMessages{}
	for event := range result.Events() {
		for _, eHandler := range r.EventHandlers {
			go eHandler.Handle(ident, u, event)
		}
		if event.Event == eventapi.EventPlaybookOnStats {
			// convert to StatusJobEvent; would love a better way to do this
			data, err := json.Marshal(event)
			if err != nil {
				printEventStats(statusEvent, u)
				return reconcile.Result{}, err
			}
			err = json.Unmarshal(data, &statusEvent)
			if err != nil {
				printEventStats(statusEvent, u)
				return reconcile.Result{}, err
			}
		}

		if module, found := event.EventData["task_action"]; found {
			if module == "operator_sdk.util.requeue_after" && event.Event != eventapi.EventRunnerOnFailed {
				if data, exists := event.EventData["res"]; exists {
					if fields, check := data.(map[string]interface{}); check {
						requeueDuration, err := time.ParseDuration(fields["period"].(string))
						if err != nil {
							logger.Error(err, "Unable to parse time input")
							return reconcileResult, err
						}
						reconcileResult.RequeueAfter = requeueDuration
						logger.Info(fmt.Sprintf("Set the reconciliation to occur after %s", requeueDuration))
						return reconcileResult, nil
					}
				}
			}
		}
		if event.Event == eventapi.EventRunnerOnFailed && !event.IgnoreError() && !event.Rescued() {
			failureMessages = append(failureMessages, event.GetFailedPlaybookMessage())
		}
	}

	// To print the stats of the task
	printEventStats(statusEvent, u)

	// To print the full ansible result
	r.printAnsibleResult(result, u)

	if statusEvent.Event == "" {
		eventErr := errors.New("did not receive playbook_on_stats event")
		stdout, err := result.Stdout()
		if err != nil {
			errmark := r.markError(ctx, request.NamespacedName, u, "Failed to get ansible-runner stdout")
			if errmark != nil {
				logger.Error(errmark, "Unable to mark error to run reconciliation")
			}
			logger.Error(err, "Failed to get ansible-runner stdout")
			return reconcileResult, err
		}
		logger.Error(eventErr, stdout)
		return reconcileResult, eventErr
	}

	// Need to get the unstructured object after the Ansible runner finishes.
	// This needs to hit the API server to retrieve updates.
	err = r.APIReader.Get(ctx, request.NamespacedName, u)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// We only want to update the CustomResource once, so we'll track changes
	// and do it at the end
	runSuccessful := len(failureMessages) == 0

	// The finalizer has run successfully, time to remove it
	deleted = u.GetDeletionTimestamp() != nil
	if deleted && finalizerExists && runSuccessful {
		controllerutil.RemoveFinalizer(u, finalizer)
		err := r.Client.Update(ctx, u)
		if err != nil {
			logger.Error(err, "Failed to remove finalizer")
			return reconcileResult, err
		}
	}
	if r.ManageStatus {
		errmark := r.markDone(ctx, request.NamespacedName, u, statusEvent, failureMessages)
		if errmark != nil {
			logger.Error(errmark, "Failed to mark status done")
		}
		// re-trigger reconcile because of failures
		if !runSuccessful {
			return reconcileResult, errors.New("event runner on failed")
		}
		return reconcileResult, errmark
	}

	// re-trigger reconcile because of failures
	if !runSuccessful {
		return reconcileResult, errors.New("received failed task event")
	}
	return reconcileResult, nil
}

func printEventStats(statusEvent eventapi.StatusJobEvent, u *unstructured.Unstructured) {
	if len(statusEvent.StdOut) > 0 {
		str := fmt.Sprintf("Ansible Task Status Event StdOut (%s, %s/%s)", u.GroupVersionKind(), u.GetName(), u.GetNamespace())
		fmt.Printf("\n----- %70s -----\n\n%s\n\n----------\n", str, statusEvent.StdOut)
	}
}

func (r *AnsibleOperatorReconciler) printAnsibleResult(result runner.RunResult, u *unstructured.Unstructured) {
	if r.AnsibleDebugLogs {
		if res, err := result.Stdout(); err == nil && len(res) > 0 {
			str := fmt.Sprintf("Ansible Debug Result (%s, %s/%s)", u.GroupVersionKind(), u.GetName(), u.GetNamespace())
			fmt.Printf("\n----- %70s -----\n\n%s\n\n----------\n", str, res)
		}
	}
}

func (r *AnsibleOperatorReconciler) markRunning(ctx context.Context, nn types.NamespacedName, u *unstructured.Unstructured) error {

	// Get the latest resource to prevent updating a stale status.
	if err := r.APIReader.Get(ctx, nn, u); err != nil {
		return err
	}
	crStatus := getStatus(u)

	// If there is no current status add that we are working on this resource.
	errCond := ansiblestatus.GetCondition(crStatus, ansiblestatus.FailureConditionType)
	if errCond != nil {
		errCond.Status = v1.ConditionFalse
		ansiblestatus.SetCondition(&crStatus, *errCond)
	}
	successCond := ansiblestatus.GetCondition(crStatus, ansiblestatus.SuccessfulConditionType)
	if successCond != nil {
		successCond.Status = v1.ConditionFalse
		ansiblestatus.SetCondition(&crStatus, *successCond)
	}
	// If the condition is currently running, making sure that the values are correct.
	// If they are the same a no-op, if they are different then it is a good thing we
	// are updating it.
	c := ansiblestatus.NewCondition(
		ansiblestatus.RunningConditionType,
		v1.ConditionTrue,
		nil,
		ansiblestatus.RunningReason,
		ansiblestatus.RunningMessage,
	)
	ansiblestatus.SetCondition(&crStatus, *c)
	u.Object["status"] = crStatus.GetJSONMap()

	return r.Client.Status().Update(ctx, u)
}

// markError - used to alert the user to the issues during the validation of a reconcile run.
// i.e Annotations that could be incorrect
func (r *AnsibleOperatorReconciler) markError(ctx context.Context, nn types.NamespacedName, u *unstructured.Unstructured,
	failureMessage string) error {

	logger := logf.Log.WithName("markError")
	// Immediately update metrics with failed reconciliation, since Get()
	// may fail.
	metrics.ReconcileFailed(r.GVK.String())
	// Get the latest resource to prevent updating a stale status.
	if err := r.APIReader.Get(ctx, nn, u); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Resource not found, assuming it was deleted")
			return nil
		}
		return err
	}
	crStatus := getStatus(u)

	rc := ansiblestatus.GetCondition(crStatus, ansiblestatus.RunningConditionType)
	if rc != nil {
		rc.Status = v1.ConditionFalse
		ansiblestatus.SetCondition(&crStatus, *rc)
	}
	sc := ansiblestatus.GetCondition(crStatus, ansiblestatus.SuccessfulConditionType)
	if sc != nil {
		sc.Status = v1.ConditionFalse
		ansiblestatus.SetCondition(&crStatus, *sc)
	}

	c := ansiblestatus.NewCondition(
		ansiblestatus.FailureConditionType,
		v1.ConditionTrue,
		nil,
		ansiblestatus.FailedReason,
		failureMessage,
	)
	ansiblestatus.SetCondition(&crStatus, *c)
	// This needs the status subresource to be enabled by default.
	u.Object["status"] = crStatus.GetJSONMap()

	return r.Client.Status().Update(ctx, u)
}

func (r *AnsibleOperatorReconciler) markDone(ctx context.Context, nn types.NamespacedName, u *unstructured.Unstructured,
	statusEvent eventapi.StatusJobEvent, failureMessages eventapi.FailureMessages) error {

	logger := logf.Log.WithName("markDone")
	// Get the latest resource to prevent updating a stale status.
	if err := r.APIReader.Get(ctx, nn, u); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Resource not found, assuming it was deleted")
			return nil
		}
		return err
	}
	crStatus := getStatus(u)

	runSuccessful := len(failureMessages) == 0
	ansibleStatus := ansiblestatus.NewAnsibleResultFromStatusJobEvent(statusEvent)

	if runSuccessful {
		metrics.ReconcileSucceeded(r.GVK.String())
		deprecatedRunningCondition := ansiblestatus.NewCondition(
			ansiblestatus.RunningConditionType,
			v1.ConditionTrue,
			ansibleStatus,
			ansiblestatus.SuccessfulReason,
			ansiblestatus.AwaitingMessage,
		)
		failureCondition := ansiblestatus.NewCondition(
			ansiblestatus.FailureConditionType,
			v1.ConditionFalse,
			nil,
			"",
			"",
		)
		successfulCondition := ansiblestatus.NewCondition(
			ansiblestatus.SuccessfulConditionType,
			v1.ConditionTrue,
			nil,
			ansiblestatus.SuccessfulReason,
			ansiblestatus.SuccessfulMessage,
		)
		ansiblestatus.SetCondition(&crStatus, *deprecatedRunningCondition)
		ansiblestatus.SetCondition(&crStatus, *successfulCondition)
		ansiblestatus.SetCondition(&crStatus, *failureCondition)
	} else {
		metrics.ReconcileFailed(r.GVK.String())
		sc := ansiblestatus.GetCondition(crStatus, ansiblestatus.RunningConditionType)
		if sc != nil {
			sc.Status = v1.ConditionFalse
			ansiblestatus.SetCondition(&crStatus, *sc)
		}
		failureCondition := ansiblestatus.NewCondition(
			ansiblestatus.FailureConditionType,
			v1.ConditionTrue,
			ansibleStatus,
			ansiblestatus.FailedReason,
			strings.Join(failureMessages, "\n"),
		)
		successfulCondition := ansiblestatus.NewCondition(
			ansiblestatus.SuccessfulConditionType,
			v1.ConditionFalse,
			nil,
			"",
			"",
		)
		ansiblestatus.SetCondition(&crStatus, *failureCondition)
		ansiblestatus.SetCondition(&crStatus, *successfulCondition)
	}
	// This needs the status subresource to be enabled by default.
	u.Object["status"] = crStatus.GetJSONMap()

	return r.Client.Status().Update(ctx, u)
}

// getStatus returns u's "status" block as a status.Status.
func getStatus(u *unstructured.Unstructured) ansiblestatus.Status {
	statusInterface := u.Object["status"]
	statusMap, ok := statusInterface.(map[string]interface{})
	// If the map is not available create one.
	if !ok {
		statusMap = map[string]interface{}{}
	}
	return ansiblestatus.CreateFromMap(statusMap)
}
