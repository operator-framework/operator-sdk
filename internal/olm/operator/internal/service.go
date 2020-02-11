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

package olm

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// withTCPPort returns a function that appends a service port to a Service's
// port list with name and TCP port portNum.
func withTCPPort(name string, portNum int32) func(*corev1.Service) {
	return func(service *corev1.Service) {
		service.Spec.Ports = append(service.Spec.Ports, corev1.ServicePort{
			Name:       name,
			Protocol:   corev1.ProtocolTCP,
			Port:       portNum,
			TargetPort: intstr.FromInt(int(portNum)),
		})
	}
}

// newRegistryService creates a new Service with a name derived from pkgName
// the package manifest's packageName, in namespace. The Service is created
// with labels derived from pkgName. opts will be applied to the Service object.
func newRegistryService(pkgName, namespace string, opts ...func(*corev1.Service)) *corev1.Service {
	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      getRegistryServerName(pkgName),
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: getRegistryDeploymentLabels(pkgName),
		},
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}
