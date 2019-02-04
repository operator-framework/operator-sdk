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

package metrics

import (
	"context"
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("metrics")

var trueVar = true

const (
	// PrometheusPortName defines the port name used in the metrics Service.
	PrometheusPortName = "metrics"
)

// ExposeMetricsPort creates a Kubernetes Service to expose the passed metrics port.
func ExposeMetricsPort(ctx context.Context, port int32) (*v1.Service, error) {
	client, err := createClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create new client: %v", err)
	}
	// We do not need to check the validity of the port, as controller-runtime
	// would error out and we would never get to this stage.
	s, err := initOperatorService(ctx, client, port, PrometheusPortName)
	if err != nil {
		if err == k8sutil.ErrNoNamespace {
			log.Info("Skipping metrics Service creation; not running in a cluster.")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to initialize service object for metrics: %v", err)
	}
	service, err := createService(ctx, client, s)
	if err != nil {
		return nil, fmt.Errorf("failed to create or get service for metrics: %v", err)
	}

	return service, nil
}

func createService(ctx context.Context, client crclient.Client, s *v1.Service) (*v1.Service, error) {
	if err := client.Create(ctx, s); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}
		// Get existing Service and return it
		existingService := &v1.Service{}
		err := client.Get(ctx, types.NamespacedName{
			Name:      s.Name,
			Namespace: s.Namespace,
		}, existingService)
		if err != nil {
			return nil, err
		}
		log.Info("Metrics Service object already exists", "name", existingService.Name)
		return existingService, nil
	}

	log.Info("Metrics Service object created", "name", s.Name)

	return s, nil
}

// initOperatorService returns the static service which exposes specifed port.
func initOperatorService(ctx context.Context, client crclient.Client, port int32, portName string) (*v1.Service, error) {
	operatorName, err := k8sutil.GetOperatorName()
	if err != nil {
		return nil, err
	}
	namespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return nil, err
	}

	label := map[string]string{"name": operatorName}

	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorName,
			Namespace: namespace,
			Labels:    label,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Port:     port,
					Protocol: v1.ProtocolTCP,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: port,
					},
					Name: portName,
				},
			},
			Selector: label,
		},
	}

	ownRef, err := getPodOwnerRef(ctx, client, namespace)
	if err != nil {
		return nil, err
	}
	service.SetOwnerReferences([]metav1.OwnerReference{*ownRef})

	return service, nil
}

func getPodOwnerRef(ctx context.Context, client crclient.Client, ns string) (*metav1.OwnerReference, error) {
	// Get current Pod the operator is running in
	pod, err := k8sutil.GetPod(ctx, client, ns)
	if err != nil {
		return nil, err
	}
	podOwnerRefs := &metav1.OwnerReference{
		APIVersion: "core/v1",
		Kind:       "Pod",
		Name:       pod.Name,
		UID:        pod.UID,
		Controller: &trueVar,
	}

	// Get Owner that the Pod belongs to
	ownerRef := metav1.GetControllerOf(pod)
	finalOwnerRef, err := findFinalOwnerRef(ctx, client, ns, ownerRef)
	if err != nil {
		return nil, err
	}
	if finalOwnerRef != nil {
		return finalOwnerRef, nil
	}

	// Default to returning Pod as the Owner
	return podOwnerRefs, nil
}

// findFinalOwnerRef tries to locate the final controller/owner based on the owner reference provided.
func findFinalOwnerRef(ctx context.Context, client crclient.Client, ns string, ownerRef *metav1.OwnerReference) (*metav1.OwnerReference, error) {
	if ownerRef == nil {
		log.V(1).Info("Pods owner could not be found")
		return nil, nil
	}

	switch ownerRef.Kind {
	case "ReplicaSet":
		// try to get the ReplicaSet owner
		rs := &appsv1.ReplicaSet{}
		key := crclient.ObjectKey{Namespace: ns, Name: ownerRef.Name}
		if err := client.Get(ctx, key, rs); err != nil {
			return nil, err
		}
		// Get Owner of the ReplicaSet and in turn the Pod belongs to
		rsOwner := metav1.GetControllerOf(rs)
		return findFinalOwnerRef(ctx, client, ns, rsOwner)
	case "DaemonSet":
		ds := &appsv1.DaemonSet{}
		key := crclient.ObjectKey{Namespace: ns, Name: ownerRef.Name}
		if err := client.Get(ctx, key, ds); err != nil {
			return nil, err
		}
		log.V(1).Info("DaemonSet was Pods owner", "DaemonSet.Name", ds.Name, "DaemonSet.Namespace", ds.Namespace)
		return &metav1.OwnerReference{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
			Name:       ds.Name,
			UID:        ds.UID,
			Controller: &trueVar,
		}, nil
	case "StatefulSet":
		ss := &appsv1.StatefulSet{}
		key := crclient.ObjectKey{Namespace: ns, Name: ownerRef.Name}
		if err := client.Get(ctx, key, ss); err != nil {
			return nil, err
		}

		log.V(1).Info("StatefulSet was Pods owner", "StatefulSet.Name", ss.Name, "StatefulSet.Namespace", ss.Namespace)
		return &metav1.OwnerReference{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
			Name:       ss.Name,
			UID:        ss.UID,
			Controller: &trueVar,
		}, nil
	case "Job":
		job := &batchv1.Job{}
		key := crclient.ObjectKey{Namespace: ns, Name: ownerRef.Name}
		if err := client.Get(ctx, key, job); err != nil {
			return nil, err
		}

		log.V(1).Info("Job was Pods owner", "Job.Name", job.Name, "Job.Namespace", job.Namespace)
		return &metav1.OwnerReference{
			APIVersion: "batch/v1",
			Kind:       "Job",
			Name:       job.Name,
			UID:        job.UID,
			Controller: &trueVar,
		}, nil
	case "Deployment":
		d := &appsv1.Deployment{}
		key := crclient.ObjectKey{Namespace: ns, Name: ownerRef.Name}
		if err := client.Get(ctx, key, d); err != nil {
			return nil, err
		}

		log.V(1).Info("Deployment was Pods owner", "Deployment.Name", d.Name, "Deployment.Namespace", d.Namespace)
		return &metav1.OwnerReference{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       d.Name,
			UID:        d.UID,
			Controller: &trueVar,
		}, nil

	case "":
		// no owner ref was found, we skip this as by default we return pod anyways
		log.V(1).Info("Pods owner could not be found, ownerRef was empty", "ownerRef.Kind", ownerRef.Kind)
	default:
		log.V(1).Info("Pods owner could not be found", "ownerRef.Kind", ownerRef.Kind)
	}

	// By default we return nothing, so later on Pod is returned.
	return nil, nil
}

func createClient() (crclient.Client, error) {
	config, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	client, err := crclient.New(config, crclient.Options{})
	if err != nil {
		return nil, err
	}

	return client, nil
}
