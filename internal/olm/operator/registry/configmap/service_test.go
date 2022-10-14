// Copyright 2020 The Operator-SDK Authors
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

package configmap

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Service", func() {
	Describe("withTCPPort", func() {
		It("should append the portnumber in the service", func() {
			ser := &corev1.Service{
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						corev1.ServicePort{
							Name: "testport",
							Port: 8000,
						},
					},
				},
			}

			x := withTCPPort("testport", 8080)
			x(ser)
			Expect(ser.Spec.Ports[1].Port).To(Equal(int32(8080)))
			Expect(ser.Spec.Ports[0].Port).To(Equal(int32(8000)))
		})
	})

	Describe("newRegistryService", func() {
		var (
			res     *corev1.Service
			service *corev1.Service
		)
		BeforeEach(func() {
			service = &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: corev1.SchemeGroupVersion.String(),
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      getRegistryServerName("pkgName"),
					Namespace: "testns",
				},
				Spec: corev1.ServiceSpec{
					Selector: getRegistryDeploymentLabels("pkgName"),
				},
			}
		})
		It("should return a service with the specified pkgName", func() {
			res = newRegistryService("pkgName", "testns", withTCPPort("testport", 8080))
			x := withTCPPort("testport", 8080)
			x(service)

			Expect(res).Should(Equal(service))
		})
	})
})
