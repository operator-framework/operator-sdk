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

package clusterserviceversion

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/operator-framework/operator-sdk/internal/generate/collector"
)

var _ = Describe("findMatchingDeploymentAndServiceForWebhook", func() {

	var (
		c   *collector.Manifests
		wcc admissionregv1.WebhookClientConfig

		depName1     = "dep-name-1"
		depName2     = "dep-name-2"
		serviceName1 = "service-name-1"
		serviceName2 = "service-name-2"
	)

	BeforeEach(func() {
		c = &collector.Manifests{}
		wcc = admissionregv1.WebhookClientConfig{}
		wcc.Service = &admissionregv1.ServiceReference{}
	})

	Context("webhook config has a matching service name", func() {
		By("parsing one deployment and one service with one label")
		It("returns the first service and deployment", func() {
			labels := map[string]string{"operator-name": "test-operator"}
			c.Deployments = []appsv1.Deployment{newDeployment(depName1, labels)}
			c.Services = []corev1.Service{newService(serviceName1, labels)}
			wcc.Service.Name = serviceName1
			depName, serviceName := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(Equal(depName1))
			Expect(serviceName).To(Equal(serviceName1))
		})

		By("parsing two deployments and two services with non-intersecting labels")
		It("returns the first service and deployment", func() {
			labels1 := map[string]string{"operator-name": "test-operator"}
			labels2 := map[string]string{"foo": "bar"}
			c.Deployments = []appsv1.Deployment{
				newDeployment(depName1, labels1),
				newDeployment(depName2, labels2),
			}
			c.Services = []corev1.Service{
				newService(serviceName1, labels1),
				newService(serviceName2, labels2),
			}
			wcc.Service.Name = serviceName1
			depName, serviceName := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(Equal(depName1))
			Expect(serviceName).To(Equal(serviceName1))
		})

		By("parsing two deployments and two services with a label subset")
		It("returns the first service and second deployment", func() {
			labels1 := map[string]string{"operator-name": "test-operator"}
			labels2 := map[string]string{"operator-name": "test-operator", "foo": "bar"}
			c.Deployments = []appsv1.Deployment{
				newDeployment(depName2, labels2),
				newDeployment(depName1, labels1),
			}
			c.Services = []corev1.Service{newService(serviceName1, labels1)}
			wcc.Service.Name = serviceName1
			depName, serviceName := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(Equal(depName2))
			Expect(serviceName).To(Equal(serviceName1))
		})
	})

	Context("webhook config does not have a matching service", func() {
		By("parsing one deployment and one service with one label")
		It("returns neither service nor deployment name", func() {
			labels := map[string]string{"operator-name": "test-operator"}
			c.Deployments = []appsv1.Deployment{newDeployment(depName1, labels)}
			c.Services = []corev1.Service{newService(serviceName1, labels)}
			wcc.Service.Name = serviceName2
			depName, serviceName := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(BeEmpty())
			Expect(serviceName).To(BeEmpty())
		})
	})

	Context("webhook config has a matching service but labels do not match", func() {
		By("parsing one deployment and one service with one label")
		It("returns the first service and no deployment", func() {
			labels1 := map[string]string{"operator-name": "test-operator"}
			labels2 := map[string]string{"foo": "bar"}
			c.Deployments = []appsv1.Deployment{newDeployment(depName1, labels1)}
			c.Services = []corev1.Service{newService(serviceName1, labels2)}
			wcc.Service.Name = serviceName1
			depName, serviceName := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(BeEmpty())
			Expect(serviceName).To(Equal(serviceName1))
		})

		By("parsing one deployment and one service with two intersecting labels")
		It("returns the first service and no deployment", func() {
			labels1 := map[string]string{"operator-name": "test-operator", "foo": "bar"}
			labels2 := map[string]string{"foo": "bar", "baz": "bat"}
			c.Deployments = []appsv1.Deployment{newDeployment(depName1, labels1)}
			c.Services = []corev1.Service{newService(serviceName1, labels2)}
			wcc.Service.Name = serviceName1
			depName, serviceName := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(BeEmpty())
			Expect(serviceName).To(Equal(serviceName1))
		})
	})
})

func newDeployment(name string, labels map[string]string) appsv1.Deployment {
	dep := appsv1.Deployment{}
	dep.SetName(name)
	dep.Spec.Template.SetLabels(labels)
	return dep
}

func newService(name string, labels map[string]string) corev1.Service {
	s := corev1.Service{}
	s.SetName(name)
	s.Spec.Selector = labels
	return s
}
