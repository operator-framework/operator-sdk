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
	"fmt"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	monclientv1 "github.com/coreos/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

var ErrServiceMonitorNotPresent = fmt.Errorf("no ServiceMonitor registered with the API")

type ServiceMonitorUpdater func(*monitoringv1.ServiceMonitor) error

// CreateServiceMonitors creates ServiceMonitors objects based on an array of Service objects.
// If CR ServiceMonitor is not registered in the Cluster it will not attempt at creating resources.
func CreateServiceMonitors(config *rest.Config, ns string, services []*v1.Service, updaters ...ServiceMonitorUpdater) ([]*monitoringv1.ServiceMonitor, error) {
	// check if we can even create ServiceMonitors
	exists, err := hasServiceMonitor(config)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrServiceMonitorNotPresent
	}

	var serviceMonitors []*monitoringv1.ServiceMonitor
	mclient := monclientv1.NewForConfigOrDie(config)

	for _, s := range services {
		if s == nil {
			continue
		}
		sm := GenerateServiceMonitor(s)
		for _, update := range updaters {
			if err := update(sm); err != nil {
				return nil, err
			}
		}

		smc, err := mclient.ServiceMonitors(ns).Create(sm)
		if err != nil {
			return serviceMonitors, err
		}
		serviceMonitors = append(serviceMonitors, smc)
	}

	return serviceMonitors, nil
}

// GenerateServiceMonitor generates a prometheus-operator ServiceMonitor object
// based on the passed Service object.
func GenerateServiceMonitor(s *v1.Service) *monitoringv1.ServiceMonitor {
	labels := make(map[string]string)
	for k, v := range s.ObjectMeta.Labels {
		labels[k] = v
	}
	endpoints := populateEndpointsFromServicePorts(s)
	boolTrue := true

	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.ObjectMeta.Name,
			Namespace: s.ObjectMeta.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v1",
					BlockOwnerDeletion: &boolTrue,
					Controller:         &boolTrue,
					Kind:               "Service",
					Name:               s.Name,
					UID:                s.UID,
				},
			},
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			Endpoints: endpoints,
		},
	}
}

func populateEndpointsFromServicePorts(s *v1.Service) []monitoringv1.Endpoint {
	var endpoints []monitoringv1.Endpoint
	for _, port := range s.Spec.Ports {
		endpoints = append(endpoints, monitoringv1.Endpoint{Port: port.Name})
	}
	return endpoints
}

// hasServiceMonitor checks if ServiceMonitor is registered in the cluster.
func hasServiceMonitor(config *rest.Config) (bool, error) {
	dc := discovery.NewDiscoveryClientForConfigOrDie(config)
	apiVersion := "monitoring.coreos.com/v1"
	kind := "ServiceMonitor"

	return k8sutil.ResourceExists(dc, apiVersion, kind)
}
