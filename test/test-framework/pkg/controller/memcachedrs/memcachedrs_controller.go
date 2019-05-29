// Copyright 2019 The Operator-SDK Authors
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

package memcachedrs

import (
	"context"
	"reflect"

	cachev1alpha1 "github.com/operator-framework/operator-sdk/test/test-framework/pkg/apis/cache/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_memcachedrs")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new MemcachedRS Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMemcachedRS{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("memcachedrs-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource MemcachedRS
	err = c.Watch(&source.Kind{Type: &cachev1alpha1.MemcachedRS{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner MemcachedRS
	err = c.Watch(&source.Kind{Type: &appsv1.ReplicaSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cachev1alpha1.MemcachedRS{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileMemcachedRS implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileMemcachedRS{}

// ReconcileMemcachedRS reconciles a MemcachedRS object
type ReconcileMemcachedRS struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a MemcachedRS object and makes changes based on the state read
// and what is in the MemcachedRS.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileMemcachedRS) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling MemcachedRS")

	// Fetch the MemcachedRS instance
	memcachedrs := &cachev1alpha1.MemcachedRS{}
	err := r.client.Get(context.TODO(), request.NamespacedName, memcachedrs)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Check if the replicaSet already exists, if not create a new one
	found := &appsv1.ReplicaSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: memcachedrs.Name, Namespace: memcachedrs.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new replicaSet
		dep := r.replicaSetForMemcached(memcachedrs)
		reqLogger.Info("Creating a new ReplicaSet", "ReplicaSet.Namespace", dep.Namespace, "ReplicaSet.Name", dep.Name)
		err = r.client.Create(context.TODO(), dep)
		if err != nil {
			reqLogger.Error(err, "Failed to create new ReplicaSet", "ReplicaSet.Namespace", dep.Namespace, "ReplicaSet.Name", dep.Name)
			return reconcile.Result{}, err
		}
		// ReplicaSet created successfully - return and requeue
		return reconcile.Result{Requeue: true}, nil
	} else if err != nil {
		reqLogger.Error(err, "Failed to get ReplicaSet")
		return reconcile.Result{}, err
	}

	// Ensure the replicaSet size is the same as the spec
	size := memcachedrs.Spec.NumNodes
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.client.Update(context.TODO(), found)
		if err != nil {
			reqLogger.Error(err, "Failed to update ReplicaSet", "ReplicaSet.Namespace", found.Namespace, "ReplicaSet.Name", found.Name)
			return reconcile.Result{}, err
		}
		// Spec updated - return and requeue
		return reconcile.Result{Requeue: true}, nil
	}

	// Update the Memcached status with the pod names
	// List the pods for this memcached's replicaSet
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(labelsForMemcached(memcachedrs.Name))
	listOps := &client.ListOptions{Namespace: memcachedrs.Namespace, LabelSelector: labelSelector}
	err = r.client.List(context.TODO(), listOps, podList)
	if err != nil {
		reqLogger.Error(err, "Failed to list pods", "Memcached.Namespace", memcachedrs.Namespace, "Memcached.Name", memcachedrs.Name)
		return reconcile.Result{}, err
	}
	podNames := getPodNames(podList.Items)

	// Update status.Nodes if needed
	if !reflect.DeepEqual(podNames, memcachedrs.Status.NodeList) {
		memcachedrs.Status.NodeList = podNames
		err := r.client.Status().Update(context.TODO(), memcachedrs)
		if err != nil {
			reqLogger.Error(err, "Failed to update Memcached status")
			return reconcile.Result{}, err
		}
	}

	// Switch testing bool
	if memcachedrs.Status.Test {
		memcachedrs.Status.Test = false
	} else {
		memcachedrs.Status.Test = true
	}
	err = r.client.Status().Update(context.TODO(), memcachedrs)
	if err != nil {
		reqLogger.Error(err, "Failed to update Memcached status")
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// rsForMemcached returns a memcached ReplicaSet object
func (r *ReconcileMemcachedRS) replicaSetForMemcached(m *cachev1alpha1.MemcachedRS) *appsv1.ReplicaSet {
	ls := labelsForMemcached(m.Name)
	replicas := m.Spec.NumNodes

	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   "memcached:1.4.36-alpine",
						Name:    "memcached",
						Command: []string{"memcached", "-m=64", "-o", "modern", "-v"},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 11211,
							Name:          "memcached",
						}},
					}},
				},
			},
		},
	}
	// Set Memcached instance as the owner and controller
	controllerutil.SetControllerReference(m, replicaSet, r.scheme)
	return replicaSet
}

// labelsForMemcached returns the labels for selecting the resources
// belonging to the given memcached CR name.
func labelsForMemcached(name string) map[string]string {
	return map[string]string{"app": "memcached-rs", "memcached_cr": name}
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
