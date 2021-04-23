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
	"errors"
	"fmt"
	"strconv"
	"time"

	rpb "helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/operator-framework/operator-sdk/internal/helm/internal/diff"
	"github.com/operator-framework/operator-sdk/internal/helm/internal/types"
	"github.com/operator-framework/operator-sdk/internal/helm/release"
)

// blank assignment to verify that HelmOperatorReconciler implements reconcile.Reconciler
var _ reconcile.Reconciler = &HelmOperatorReconciler{}

// ReleaseHookFunc defines a function signature for release hooks.
type ReleaseHookFunc func(*rpb.Release) error

// HelmOperatorReconciler reconciles custom resources as Helm releases.
type HelmOperatorReconciler struct {
	Client          client.Client
	EventRecorder   record.EventRecorder
	GVK             schema.GroupVersionKind
	ManagerFactory  release.ManagerFactory
	ReconcilePeriod time.Duration
	OverrideValues  map[string]string
	releaseHook     ReleaseHookFunc
}

const (
	// uninstallFinalizer is added to CRs so they are cleaned up after uninstalling a release.
	uninstallFinalizer = "helm.sdk.operatorframework.io/uninstall-release"
	// Deprecated: use uninstallFinalizer. This will be removed in operator-sdk v2.0.0.
	uninstallFinalizerLegacy = "uninstall-helm-release"

	helmUpgradeForceAnnotation  = "helm.sdk.operatorframework.io/upgrade-force"
	helmUninstallWaitAnnotation = "helm.sdk.operatorframework.io/uninstall-wait"
)

// Reconcile reconciles the requested resource by installing, updating, or
// uninstalling a Helm release based on the resource's current state. If no
// release changes are necessary, Reconcile will create or patch the underlying
// resources to match the expected release manifest.

func (r HelmOperatorReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) { //nolint:gocyclo
	o := &unstructured.Unstructured{}
	o.SetGroupVersionKind(r.GVK)
	o.SetNamespace(request.Namespace)
	o.SetName(request.Name)
	log := log.WithValues(
		"namespace", o.GetNamespace(),
		"name", o.GetName(),
		"apiVersion", o.GetAPIVersion(),
		"kind", o.GetKind(),
	)
	log.V(1).Info("Reconciling")

	err := r.Client.Get(ctx, request.NamespacedName, o)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}
	if err != nil {
		log.Error(err, "Failed to lookup resource")
		return reconcile.Result{}, err
	}

	manager, err := r.ManagerFactory.NewManager(o, r.OverrideValues)
	if err != nil {
		log.Error(err, "Failed to get release manager")
		return reconcile.Result{}, err
	}

	status := types.StatusFor(o)
	log = log.WithValues("release", manager.ReleaseName())

	if o.GetDeletionTimestamp() != nil {
		if !(controllerutil.ContainsFinalizer(o, uninstallFinalizer) ||
			controllerutil.ContainsFinalizer(o, uninstallFinalizerLegacy)) {

			log.Info("Resource is terminated, skipping reconciliation")
			return reconcile.Result{}, nil
		}

		uninstalledRelease, err := manager.UninstallRelease(ctx)
		if err != nil && !errors.Is(err, driver.ErrReleaseNotFound) {
			log.Error(err, "Failed to uninstall release")
			status.SetCondition(types.HelmAppCondition{
				Type:    types.ConditionReleaseFailed,
				Status:  types.StatusTrue,
				Reason:  types.ReasonUninstallError,
				Message: err.Error(),
			})
			if err := r.updateResourceStatus(ctx, o, status); err != nil {
				log.Error(err, "Failed to update status after uninstall release failure")
			}
			return reconcile.Result{}, err
		}
		status.RemoveCondition(types.ConditionReleaseFailed)

		wait := hasAnnotation(helmUninstallWaitAnnotation, o)
		if errors.Is(err, driver.ErrReleaseNotFound) {
			log.Info("Release not found")
		} else {
			log.Info("Uninstalled release")
			if log.V(0).Enabled() && uninstalledRelease != nil {
				fmt.Println(diff.Generate(uninstalledRelease.Manifest, ""))
			}
			if !wait {
				status.SetCondition(types.HelmAppCondition{
					Type:   types.ConditionDeployed,
					Status: types.StatusFalse,
					Reason: types.ReasonUninstallSuccessful,
				})
				status.DeployedRelease = nil
			}
		}
		if wait {
			status.SetCondition(types.HelmAppCondition{
				Type:    types.ConditionDeployed,
				Status:  types.StatusFalse,
				Reason:  types.ReasonUninstallSuccessful,
				Message: "Waiting until all resources are deleted.",
			})
		}
		if err := r.updateResourceStatus(ctx, o, status); err != nil {
			log.Info("Failed to update CR status")
			return reconcile.Result{}, err
		}

		if wait && status.DeployedRelease != nil && status.DeployedRelease.Manifest != "" {
			log.Info("Uninstall wait")
			isAllResourcesDeleted, err := manager.CleanupRelease(ctx, status.DeployedRelease.Manifest)
			if err != nil {
				log.Error(err, "Failed to cleanup release")
				status.SetCondition(types.HelmAppCondition{
					Type:    types.ConditionReleaseFailed,
					Status:  types.StatusTrue,
					Reason:  types.ReasonUninstallError,
					Message: err.Error(),
				})
				_ = r.updateResourceStatus(ctx, o, status)
				return reconcile.Result{}, err
			}
			if !isAllResourcesDeleted {
				log.Info("Waiting until all resources are deleted")
				return reconcile.Result{RequeueAfter: r.ReconcilePeriod}, nil
			}
			status.RemoveCondition(types.ConditionReleaseFailed)
		}

		log.Info("Removing finalizer")
		controllerutil.RemoveFinalizer(o, uninstallFinalizer)
		controllerutil.RemoveFinalizer(o, uninstallFinalizerLegacy)
		if err := r.updateResource(ctx, o); err != nil {
			log.Info("Failed to remove CR uninstall finalizer")
			return reconcile.Result{}, err
		}

		// Since the client is hitting a cache, waiting for the
		// deletion here will guarantee that the next reconciliation
		// will see that the CR has been deleted and that there's
		// nothing left to do.
		if err := r.waitForDeletion(ctx, o); err != nil {
			log.Info("Failed waiting for CR deletion")
			return reconcile.Result{}, err
		}

		return reconcile.Result{}, nil
	}

	status.SetCondition(types.HelmAppCondition{
		Type:   types.ConditionInitialized,
		Status: types.StatusTrue,
	})

	if err := manager.Sync(ctx); err != nil {
		log.Error(err, "Failed to sync release")
		status.SetCondition(types.HelmAppCondition{
			Type:    types.ConditionIrreconcilable,
			Status:  types.StatusTrue,
			Reason:  types.ReasonReconcileError,
			Message: err.Error(),
		})
		if err := r.updateResourceStatus(ctx, o, status); err != nil {
			log.Error(err, "Failed to update status after sync release failure")
		}
		return reconcile.Result{}, err
	}
	status.RemoveCondition(types.ConditionIrreconcilable)

	if !manager.IsInstalled() {
		for k, v := range r.OverrideValues {
			r.EventRecorder.Eventf(o, "Warning", "OverrideValuesInUse",
				"Chart value %q overridden to %q by operator's watches.yaml", k, v)
		}
		installedRelease, err := manager.InstallRelease(ctx)
		if err != nil {
			log.Error(err, "Release failed")
			status.SetCondition(types.HelmAppCondition{
				Type:    types.ConditionReleaseFailed,
				Status:  types.StatusTrue,
				Reason:  types.ReasonInstallError,
				Message: err.Error(),
			})
			if err := r.updateResourceStatus(ctx, o, status); err != nil {
				log.Error(err, "Failed to update status after install release failure")
			}
			return reconcile.Result{}, err
		}
		status.RemoveCondition(types.ConditionReleaseFailed)

		log.V(1).Info("Adding finalizer", "finalizer", uninstallFinalizer)
		controllerutil.AddFinalizer(o, uninstallFinalizer)
		if err := r.updateResource(ctx, o); err != nil {
			log.Info("Failed to add CR uninstall finalizer")
			return reconcile.Result{}, err
		}

		if r.releaseHook != nil {
			if err := r.releaseHook(installedRelease); err != nil {
				log.Error(err, "Failed to run release hook")
				return reconcile.Result{}, err
			}
		}

		log.Info("Installed release")
		if log.V(0).Enabled() {
			fmt.Println(diff.Generate("", installedRelease.Manifest))
		}
		log.V(1).Info("Config values", "values", installedRelease.Config)
		message := ""
		if installedRelease.Info != nil {
			message = installedRelease.Info.Notes
		}
		status.SetCondition(types.HelmAppCondition{
			Type:    types.ConditionDeployed,
			Status:  types.StatusTrue,
			Reason:  types.ReasonInstallSuccessful,
			Message: message,
		})
		status.DeployedRelease = &types.HelmAppRelease{
			Name:     installedRelease.Name,
			Manifest: installedRelease.Manifest,
		}
		err = r.updateResourceStatus(ctx, o, status)
		return reconcile.Result{RequeueAfter: r.ReconcilePeriod}, err
	}

	if !(controllerutil.ContainsFinalizer(o, uninstallFinalizer) ||
		controllerutil.ContainsFinalizer(o, uninstallFinalizerLegacy)) {

		log.V(1).Info("Adding finalizer", "finalizer", uninstallFinalizer)
		controllerutil.AddFinalizer(o, uninstallFinalizer)
		if err := r.updateResource(ctx, o); err != nil {
			log.Info("Failed to add CR uninstall finalizer")
			return reconcile.Result{}, err
		}
	}

	if manager.IsUpgradeRequired() {
		for k, v := range r.OverrideValues {
			r.EventRecorder.Eventf(o, "Warning", "OverrideValuesInUse",
				"Chart value %q overridden to %q by operator's watches.yaml", k, v)
		}
		force := hasAnnotation(helmUpgradeForceAnnotation, o)
		previousRelease, upgradedRelease, err := manager.UpgradeRelease(ctx, release.ForceUpgrade(force))
		if err != nil {
			log.Error(err, "Release failed")
			status.SetCondition(types.HelmAppCondition{
				Type:    types.ConditionReleaseFailed,
				Status:  types.StatusTrue,
				Reason:  types.ReasonUpgradeError,
				Message: err.Error(),
			})
			if err := r.updateResourceStatus(ctx, o, status); err != nil {
				log.Error(err, "Failed to update status after sync release failure")
			}
			return reconcile.Result{}, err
		}
		status.RemoveCondition(types.ConditionReleaseFailed)

		if r.releaseHook != nil {
			if err := r.releaseHook(upgradedRelease); err != nil {
				log.Error(err, "Failed to run release hook")
				return reconcile.Result{}, err
			}
		}

		log.Info("Upgraded release", "force", force)
		if log.V(0).Enabled() {
			fmt.Println(diff.Generate(previousRelease.Manifest, upgradedRelease.Manifest))
		}
		log.V(1).Info("Config values", "values", upgradedRelease.Config)
		message := ""
		if upgradedRelease.Info != nil {
			message = upgradedRelease.Info.Notes
		}
		status.SetCondition(types.HelmAppCondition{
			Type:    types.ConditionDeployed,
			Status:  types.StatusTrue,
			Reason:  types.ReasonUpgradeSuccessful,
			Message: message,
		})
		status.DeployedRelease = &types.HelmAppRelease{
			Name:     upgradedRelease.Name,
			Manifest: upgradedRelease.Manifest,
		}
		err = r.updateResourceStatus(ctx, o, status)
		return reconcile.Result{RequeueAfter: r.ReconcilePeriod}, err
	}

	// If a change is made to the CR spec that causes a release failure, a
	// ConditionReleaseFailed is added to the status conditions. If that change
	// is then reverted to its previous state, the operator will stop
	// attempting the release and will resume reconciling. In this case, we
	// need to remove the ConditionReleaseFailed because the failing release is
	// no longer being attempted.
	status.RemoveCondition(types.ConditionReleaseFailed)

	expectedRelease, err := manager.ReconcileRelease(ctx)
	if err != nil {
		log.Error(err, "Failed to reconcile release")
		status.SetCondition(types.HelmAppCondition{
			Type:    types.ConditionIrreconcilable,
			Status:  types.StatusTrue,
			Reason:  types.ReasonReconcileError,
			Message: err.Error(),
		})
		if err := r.updateResourceStatus(ctx, o, status); err != nil {
			log.Error(err, "Failed to update status after reconcile release failure")
		}
		return reconcile.Result{}, err
	}
	status.RemoveCondition(types.ConditionIrreconcilable)

	if r.releaseHook != nil {
		if err := r.releaseHook(expectedRelease); err != nil {
			log.Error(err, "Failed to run release hook")
			return reconcile.Result{}, err
		}
	}

	log.Info("Reconciled release")
	reason := types.ReasonUpgradeSuccessful
	if expectedRelease.Version == 1 {
		reason = types.ReasonInstallSuccessful
	}
	message := ""
	if expectedRelease.Info != nil {
		message = expectedRelease.Info.Notes
	}
	status.SetCondition(types.HelmAppCondition{
		Type:    types.ConditionDeployed,
		Status:  types.StatusTrue,
		Reason:  reason,
		Message: message,
	})
	status.DeployedRelease = &types.HelmAppRelease{
		Name:     expectedRelease.Name,
		Manifest: expectedRelease.Manifest,
	}
	err = r.updateResourceStatus(ctx, o, status)
	return reconcile.Result{RequeueAfter: r.ReconcilePeriod}, err
}

// returns the boolean representation of the annotation string
// will return false if annotation is not set
func hasAnnotation(anno string, o *unstructured.Unstructured) bool {
	boolStr := o.GetAnnotations()[anno]
	if boolStr == "" {
		return false
	}
	value := false
	if i, err := strconv.ParseBool(boolStr); err != nil {
		log.Info("Could not parse annotation as a boolean",
			"annotation", anno, "value informed", boolStr)
	} else {
		value = i
	}
	return value
}

func (r HelmOperatorReconciler) updateResource(ctx context.Context, o client.Object) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		return r.Client.Update(ctx, o)
	})
}

func (r HelmOperatorReconciler) updateResourceStatus(ctx context.Context, o *unstructured.Unstructured, status *types.HelmAppStatus) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		o.Object["status"] = status
		return r.Client.Status().Update(ctx, o)
	})
}

func (r HelmOperatorReconciler) waitForDeletion(ctx context.Context, o client.Object) error {
	key := client.ObjectKeyFromObject(o)

	tctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()
	return wait.PollImmediateUntil(time.Millisecond*10, func() (bool, error) {
		err := r.Client.Get(tctx, key, o)
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		return false, nil
	}, tctx.Done())
}
