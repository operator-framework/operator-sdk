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
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/operator-framework/operator-sdk/internal/util/diffutil"
	"github.com/operator-framework/operator-sdk/pkg/helm/internal/types"
	"github.com/operator-framework/operator-sdk/pkg/helm/release"
)

var _ reconcile.Reconciler = &HelmOperatorReconciler{}

// HelmOperatorReconciler reconciles custom resources as Helm releases.
type HelmOperatorReconciler struct {
	Client         client.Client
	GVK            schema.GroupVersionKind
	ManagerFactory release.ManagerFactory
	ResyncPeriod   time.Duration
}

const (
	finalizer = "uninstall-helm-release"
)

// Reconcile reconciles the requested resource by installing, updating, or
// uninstalling a Helm release based on the resource's current state. If no
// release changes are necessary, Reconcile will create or patch the underlying
// resources to match the expected release manifest.
func (r HelmOperatorReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
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

	err := r.Client.Get(context.TODO(), request.NamespacedName, o)
	if apierrors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}
	if err != nil {
		log.Error(err, "failed to lookup resource")
		return reconcile.Result{}, err
	}

	manager := r.ManagerFactory.NewManager(o)
	status := types.StatusFor(o)
	log = log.WithValues("release", manager.ReleaseName())

	deleted := o.GetDeletionTimestamp() != nil
	pendingFinalizers := o.GetFinalizers()
	if !deleted && !contains(pendingFinalizers, finalizer) {
		log.V(1).Info("Adding finalizer", "finalizer", finalizer)
		finalizers := append(pendingFinalizers, finalizer)
		o.SetFinalizers(finalizers)
		err := r.Client.Update(context.TODO(), o)
		return reconcile.Result{}, err
	}

	if err := manager.Sync(context.TODO()); err != nil {
		log.Error(err, "failed to sync release")
		return reconcile.Result{}, err
	}

	if deleted {
		if !contains(pendingFinalizers, finalizer) {
			log.Info("Resource is terminated, skipping reconciliation")
			return reconcile.Result{}, nil
		}

		uninstalledRelease, err := manager.UninstallRelease(context.TODO())
		if err != nil && err != release.ErrNotFound {
			log.Error(err, "failed to uninstall release")
			return reconcile.Result{}, err
		}
		if err == release.ErrNotFound {
			log.Info("Release not found, removing finalizer")
		} else {
			log.Info("Uninstalled release")
			if log.Enabled() {
				fmt.Println(diffutil.Diff(uninstalledRelease.GetManifest(), ""))
			}
		}
		finalizers := []string{}
		for _, pendingFinalizer := range pendingFinalizers {
			if pendingFinalizer != finalizer {
				finalizers = append(finalizers, pendingFinalizer)
			}
		}
		o.SetFinalizers(finalizers)
		err = r.Client.Update(context.TODO(), o)
		return reconcile.Result{}, err
	}

	if !manager.IsInstalled() {
		installedRelease, err := manager.InstallRelease(context.TODO())
		if err != nil {
			log.Error(err, "failed to install release")
			return reconcile.Result{}, err
		}

		log.Info("Installed release")
		if log.Enabled() {
			fmt.Println(diffutil.Diff("", installedRelease.GetManifest()))
		}
		log.V(1).Info("Config values", "values", installedRelease.GetConfig())
		status.SetRelease(installedRelease)
		status.SetPhase(types.PhaseApplied, types.ReasonApplySuccessful, installedRelease.GetInfo().GetStatus().GetNotes())
		err = r.updateResourceStatus(o, status)
		return reconcile.Result{RequeueAfter: r.ResyncPeriod}, err
	}

	if manager.IsUpdateRequired() {
		previousRelease, updatedRelease, err := manager.UpdateRelease(context.TODO())
		if err != nil {
			log.Error(err, "failed to update release")
			return reconcile.Result{}, err
		}
		log.Info("Updated release")
		if log.Enabled() {
			fmt.Println(diffutil.Diff(previousRelease.GetManifest(), updatedRelease.GetManifest()))
		}
		log.V(1).Info("Config values", "values", updatedRelease.GetConfig())
		status.SetRelease(updatedRelease)
		status.SetPhase(types.PhaseApplied, types.ReasonApplySuccessful, updatedRelease.GetInfo().GetStatus().GetNotes())
		err = r.updateResourceStatus(o, status)
		return reconcile.Result{RequeueAfter: r.ResyncPeriod}, err
	}

	_, err = manager.ReconcileRelease(context.TODO())
	if err != nil {
		log.Error(err, "failed to reconcile release")
		return reconcile.Result{}, err
	}

	log.Info("Reconciled release")
	return reconcile.Result{RequeueAfter: r.ResyncPeriod}, nil
}

func (r HelmOperatorReconciler) updateResourceStatus(o *unstructured.Unstructured, status *types.HelmAppStatus) error {
	o.Object["status"] = status
	return r.Client.Status().Update(context.TODO(), o)
}

func contains(l []string, s string) bool {
	for _, elem := range l {
		if elem == s {
			return true
		}
	}
	return false
}
