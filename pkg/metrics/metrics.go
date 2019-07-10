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

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("metrics")

var trueVar = true

const (
	// OperatorPortName defines the default operator metrics port name used in the metrics Service.
	OperatorPortName = "http-metrics"
	// CRPortName defines the custom resource specific metrics' port name used in the metrics Service.
	CRPortName = "cr-metrics"
)

// CreateMetricsService creates a Kubernetes Service to expose the passed metrics
// port(s) with the given name(s).
func CreateMetricsService(ctx context.Context, cfg *rest.Config, servicePorts []v1.ServicePort) (*v1.Service, error) {
	if len(servicePorts) < 1 {
		return nil, fmt.Errorf("failed to create metrics Serice; service ports were empty")
	}
	client, err := crclient.New(cfg, crclient.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create new client: %v", err)
	}
	s, err := initOperatorService(ctx, client, servicePorts)
	if err != nil {
		if err == k8sutil.ErrNoNamespace {
			log.Info("Skipping metrics Service creation; not running in a cluster.")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to initialize service object for metrics: %v", err)
	}
	service, err := createOrUpdateService(ctx, client, s)
	if err != nil {
		return nil, fmt.Errorf("failed to create or get service for metrics: %v", err)
	}

	return service, nil
}

func createOrUpdateService(ctx context.Context, client crclient.Client, s *v1.Service) (*v1.Service, error) {
	if err := client.Create(ctx, s); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return nil, err
		}
		// Service already exists, we want to update it
		// as we do not know if any fields might have changed.
		existingService := &v1.Service{}
		err := client.Get(ctx, types.NamespacedName{
			Name:      s.Name,
			Namespace: s.Namespace,
		}, existingService)

		s.ResourceVersion = existingService.ResourceVersion
		if existingService.Spec.Type == v1.ServiceTypeClusterIP {
			s.Spec.ClusterIP = existingService.Spec.ClusterIP
		}
		err = client.Update(ctx, s)
		if err != nil {
			return nil, err
		}
		log.V(1).Info("Metrics Service object updated", "Service.Name", s.Name, "Service.Namespace", s.Namespace)
		return existingService, nil
	}

	log.Info("Metrics Service object created", "Service.Name", s.Name, "Service.Namespace", s.Namespace)
	return s, nil
}

// initOperatorService returns the static service which exposes specified port(s).
func initOperatorService(ctx context.Context, client crclient.Client, sp []v1.ServicePort) (*v1.Service, error) {
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
			Name:      fmt.Sprintf("%s-metrics", operatorName),
			Namespace: namespace,
			Labels:    label,
		},
		Spec: v1.ServiceSpec{
			Ports:    sp,
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
	podOwnerRefs := metav1.NewControllerRef(pod, pod.GroupVersionKind())
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
		return nil, nil
	}

	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(ownerRef.APIVersion)
	obj.SetKind(ownerRef.Kind)
	err := client.Get(ctx, types.NamespacedName{Namespace: ns, Name: ownerRef.Name}, obj)
	if err != nil {
		return nil, err
	}
	newOwnerRef := metav1.GetControllerOf(obj)
	if newOwnerRef != nil {
		return findFinalOwnerRef(ctx, client, ns, newOwnerRef)
	}

	log.V(1).Info("Pods owner found", "Kind", ownerRef.Kind, "Name", ownerRef.Name, "Namespace", ns)
	return ownerRef, nil
}
