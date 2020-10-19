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
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/operator-framework/operator-sdk/internal/generate/collector"
)

var _ = Describe("apply functions", func() {
	var (
		c        *collector.Manifests
		strategy *operatorsv1alpha1.StrategyDetailsDeployment
	)

	Describe("apply{Cluster}Roles", func() {
		const (
			depName1   = "dep-1"
			saName1    = "service-account-1"
			roleName1  = "role-1"
			cRoleName1 = "cluster-role-1"
		)

		BeforeEach(func() {
			c = &collector.Manifests{}
			strategy = &operatorsv1alpha1.StrategyDetailsDeployment{}
		})

		Context("collector contains Roles", func() {
			It("adds one Role's rules to the CSV deployment strategy", func() {
				c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount(depName1, saName1)}
				c.ServiceAccounts = []corev1.ServiceAccount{newServiceAccount(saName1)}
				rules := []rbacv1.PolicyRule{{Verbs: []string{"create"}}}
				c.Roles = []rbacv1.Role{newRole(roleName1, rules...)}
				c.RoleBindings = []rbacv1.RoleBinding{newRoleBinding("role-binding", newRoleRef(roleName1), newServiceAccountSubject(saName1))}
				applyRoles(c, strategy)
				Expect(strategy.Permissions).To(Equal([]operatorsv1alpha1.StrategyDeploymentPermissions{
					{ServiceAccountName: saName1, Rules: rules},
				}))
			})
		})
		Context("collector contains no Roles", func() {
			It("adds no Permissions to the CSV deployment strategy", func() {
				c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount(depName1, saName1)}
				c.ServiceAccounts = []corev1.ServiceAccount{newServiceAccount(saName1)}
				c.RoleBindings = []rbacv1.RoleBinding{newRoleBinding("role-binding", newRoleRef(roleName1), newServiceAccountSubject(saName1))}
				applyRoles(c, strategy)
				Expect(strategy.Permissions).To(Equal([]operatorsv1alpha1.StrategyDeploymentPermissions{}))
			})
		})

		Context("collector contains ClusterRoles", func() {
			It("adds one ClusterRole's rules to the CSV deployment strategy", func() {
				c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount(depName1, saName1)}
				c.ServiceAccounts = []corev1.ServiceAccount{newServiceAccount(saName1)}
				rules := []rbacv1.PolicyRule{{Verbs: []string{"create"}}}
				c.ClusterRoles = []rbacv1.ClusterRole{newClusterRole(cRoleName1, rules...)}
				c.ClusterRoleBindings = []rbacv1.ClusterRoleBinding{newClusterRoleBinding("cluster-role-binding", newClusterRoleRef(cRoleName1), newServiceAccountSubject(saName1))}
				applyClusterRoles(c, strategy)
				Expect(strategy.ClusterPermissions).To(Equal([]operatorsv1alpha1.StrategyDeploymentPermissions{
					{ServiceAccountName: saName1, Rules: rules},
				}))
			})
		})
		Context("collector contains no ClusterRoles", func() {
			It("adds no ClusterPermissions to the CSV deployment strategy", func() {
				c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount(depName1, saName1)}
				c.ServiceAccounts = []corev1.ServiceAccount{newServiceAccount(saName1)}
				c.ClusterRoleBindings = []rbacv1.ClusterRoleBinding{newClusterRoleBinding("cluster-role-binding", newClusterRoleRef(cRoleName1), newServiceAccountSubject(saName1))}
				applyClusterRoles(c, strategy)
				Expect(strategy.ClusterPermissions).To(Equal([]operatorsv1alpha1.StrategyDeploymentPermissions{}))
			})
		})
	})
})

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
			depName, service := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(Equal(depName1))
			Expect(service.GetName()).To(Equal(serviceName1))
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
			depName, service := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(Equal(depName1))
			Expect(service.GetName()).To(Equal(serviceName1))
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
			depName, service := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(Equal(depName2))
			Expect(service.GetName()).To(Equal(serviceName1))
		})
	})

	Context("webhook config does not have a matching service", func() {
		By("parsing one deployment and one service with one label")
		It("returns neither service nor deployment name", func() {
			labels := map[string]string{"operator-name": "test-operator"}
			c.Deployments = []appsv1.Deployment{newDeployment(depName1, labels)}
			c.Services = []corev1.Service{newService(serviceName1, labels)}
			wcc.Service.Name = serviceName2
			depName, service := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(BeEmpty())
			Expect(service).To(BeNil())
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
			depName, service := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(BeEmpty())
			Expect(service.GetName()).To(Equal(serviceName1))
		})

		By("parsing one deployment and one service with two intersecting labels")
		It("returns the first service and no deployment", func() {
			labels1 := map[string]string{"operator-name": "test-operator", "foo": "bar"}
			labels2 := map[string]string{"foo": "bar", "baz": "bat"}
			c.Deployments = []appsv1.Deployment{newDeployment(depName1, labels1)}
			c.Services = []corev1.Service{newService(serviceName1, labels2)}
			wcc.Service.Name = serviceName1
			depName, service := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(BeEmpty())
			Expect(service.GetName()).To(Equal(serviceName1))
		})
	})
})

func newDeployment(name string, labels map[string]string) (dep appsv1.Deployment) {
	dep.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	dep.SetName(name)
	dep.Spec.Template.SetLabels(labels)
	return dep
}

func newService(name string, labels map[string]string) (s corev1.Service) {
	s.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
	s.SetName(name)
	s.Spec.Selector = labels
	return s
}

//nolint:unparam
func newServiceAccount(name string) (s corev1.ServiceAccount) {
	s.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ServiceAccount"))
	s.SetName(name)
	return s
}

//nolint:unparam
func newDeploymentWithServiceAccount(name, saName string) (d appsv1.Deployment) {
	d = newDeployment(name, nil)
	d.Spec.Template.Spec.ServiceAccountName = saName
	return d
}

func newRole(name string, rules ...rbacv1.PolicyRule) (r rbacv1.Role) {
	r.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("Role"))
	r.SetName(name)
	r.Rules = rules
	return r
}

func newClusterRole(name string, rules ...rbacv1.PolicyRule) (r rbacv1.ClusterRole) {
	r.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("ClusterRole"))
	r.SetName(name)
	r.Rules = rules
	return r
}

func newRoleBinding(name string, ref rbacv1.RoleRef, subjects ...rbacv1.Subject) (r rbacv1.RoleBinding) {
	r.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("RoleBinding"))
	r.SetName(name)
	r.RoleRef = ref
	r.Subjects = subjects
	return r
}

func newClusterRoleBinding(name string, ref rbacv1.RoleRef, subjects ...rbacv1.Subject) (r rbacv1.ClusterRoleBinding) {
	r.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding"))
	r.SetName(name)
	r.RoleRef = ref
	r.Subjects = subjects
	return r
}

func newRef(name, kind, apiGroup string) (s rbacv1.RoleRef) {
	s.Name = name
	s.Kind = kind
	s.APIGroup = apiGroup
	return s
}

func newRoleRef(name string) (s rbacv1.RoleRef) {
	return newRef(name, "Role", rbacv1.SchemeGroupVersion.Group)
}

func newClusterRoleRef(name string) (s rbacv1.RoleRef) {
	return newRef(name, "ClusterRole", rbacv1.SchemeGroupVersion.Group)
}

func newSubject(name, kind string) (s rbacv1.Subject) {
	s.Name = name
	s.Kind = kind
	return s
}

//nolint:unparam
func newServiceAccountSubject(name string) rbacv1.Subject {
	return newSubject(name, "ServiceAccount")
}
