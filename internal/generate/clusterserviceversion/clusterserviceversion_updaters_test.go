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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/operator-sdk/internal/generate/collector"
)

var _ = Describe("apply functions", func() {
	var (
		c        *collector.Manifests
		strategy *operatorsv1alpha1.StrategyDetailsDeployment
	)

	Describe("applyDeployments", func() {
		const (
			depName = "dep-1"
		)

		BeforeEach(func() {
			c = &collector.Manifests{}
			strategy = &operatorsv1alpha1.StrategyDetailsDeployment{}
		})

		Context("collector contains Deployments", func() {
			It("applies the deployment labels", func() {
				labels := labels.Set{}
				labels["foo"] = "bar"

				c.Deployments = []appsv1.Deployment{newDeploymentWithLabels(depName, labels)}
				applyDeployments(c, strategy)
				Expect(strategy.DeploymentSpecs).To(HaveLen(1))
				Expect(strategy.DeploymentSpecs[0].Label).To(Equal(labels))
			})
		})
	})

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

		Context("collector contains {Cluster}Roles", func() {
			It("adds one Role's rules to the CSV deployment strategy", func() {
				c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount(depName1, saName1)}
				c.ServiceAccounts = []corev1.ServiceAccount{newServiceAccount(saName1)}
				rules := []rbacv1.PolicyRule{{Verbs: []string{"create"}}}
				perms := []client.Object{newRole(roleName1, rules...)}
				c.RoleBindings = []rbacv1.RoleBinding{newRoleBinding("role-binding", newRoleRef(roleName1), newServiceAccountSubject(saName1))}
				applyRoles(c, perms, strategy, nil)
				Expect(strategy.Permissions).To(Equal([]operatorsv1alpha1.StrategyDeploymentPermissions{
					{ServiceAccountName: saName1, Rules: rules},
				}))
			})
			It("adds one ClusterRole's rules to the CSV deployment strategy", func() {
				c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount(depName1, saName1)}
				c.ServiceAccounts = []corev1.ServiceAccount{newServiceAccount(saName1)}
				rules := []rbacv1.PolicyRule{{Verbs: []string{"create"}}}
				perms := []client.Object{newClusterRole(cRoleName1, rules...)}
				c.ClusterRoleBindings = []rbacv1.ClusterRoleBinding{newClusterRoleBinding("cluster-role-binding", newClusterRoleRef(cRoleName1), newServiceAccountSubject(saName1))}
				applyClusterRoles(c, perms, strategy, nil)
				Expect(strategy.ClusterPermissions).To(Equal([]operatorsv1alpha1.StrategyDeploymentPermissions{
					{ServiceAccountName: saName1, Rules: rules},
				}))
			})
			It("adds multiple bound {Cluster}Roles to the CSV deployment strategy with extra service account", func() {
				roleName2, roleName3 := "role-2", "role-3"
				cRoleName2, cRoleName3 := "cluster-role-2", "cluster-role-3"
				extraSAName := "service-account-extra"
				c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount(depName1, saName1)}
				c.ServiceAccounts = []corev1.ServiceAccount{
					newServiceAccount(saName1),
					newServiceAccount(extraSAName),
				}
				rules := []rbacv1.PolicyRule{{Verbs: []string{"create"}}}
				role3Rules := []rbacv1.PolicyRule{{APIGroups: []string{"my.group"}, Verbs: []string{"update"}}}
				cRole3Rules := []rbacv1.PolicyRule{{APIGroups: []string{"my.group"}, Verbs: []string{"list", "watch"}}}
				perms := []client.Object{
					newRole(roleName1, rules...),
					newRole(roleName2, rules...),
					newRole(roleName3, role3Rules...),
					newClusterRole(cRoleName1, rules...),
					newClusterRole(cRoleName2, rules...),
				}
				cperms := []client.Object{
					newClusterRole(cRoleName1, rules...),
					newClusterRole(cRoleName3, cRole3Rules...),
				}
				c.RoleBindings = []rbacv1.RoleBinding{
					newRoleBinding("role-binding", newRoleRef(roleName1), newServiceAccountSubject(saName1)),
					newRoleBinding("role-binding-2", newRoleRef(roleName2), newServiceAccountSubject(extraSAName)),
					newRoleBinding("role-binding-3", newClusterRoleRef(cRoleName3), newServiceAccountSubject(extraSAName)),
				}
				c.ClusterRoleBindings = []rbacv1.ClusterRoleBinding{
					newClusterRoleBinding("cluster-role-binding", newClusterRoleRef(cRoleName1), newServiceAccountSubject(saName1)),
					newClusterRoleBinding("cluster-role-binding-2", newClusterRoleRef(cRoleName2), newServiceAccountSubject(extraSAName)),
					newClusterRoleBinding("cluster-role-binding-3", newClusterRoleRef(cRoleName3), newServiceAccountSubject(extraSAName)),
				}
				applyRoles(c, perms, strategy, []string{extraSAName})
				applyClusterRoles(c, cperms, strategy, []string{extraSAName})
				Expect(strategy.Permissions).To(Equal([]operatorsv1alpha1.StrategyDeploymentPermissions{
					{ServiceAccountName: saName1, Rules: rules},
					{ServiceAccountName: extraSAName, Rules: rules},
				}))
				Expect(strategy.ClusterPermissions).To(Equal([]operatorsv1alpha1.StrategyDeploymentPermissions{
					{ServiceAccountName: saName1, Rules: rules},
					{ServiceAccountName: extraSAName, Rules: cRole3Rules},
				}))
			})
		})

		Context("collector contains no {Cluster}Roles", func() {
			It("adds no Permissions to the CSV deployment strategy", func() {
				c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount(depName1, saName1)}
				c.ServiceAccounts = []corev1.ServiceAccount{newServiceAccount(saName1)}
				c.RoleBindings = []rbacv1.RoleBinding{newRoleBinding("role-binding", newRoleRef(roleName1), newServiceAccountSubject(saName1))}
				applyRoles(c, nil, strategy, nil)
				Expect(strategy.Permissions).To(Equal([]operatorsv1alpha1.StrategyDeploymentPermissions{}))
			})
			It("adds no ClusterPermissions to the CSV deployment strategy", func() {
				c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount(depName1, saName1)}
				c.ServiceAccounts = []corev1.ServiceAccount{newServiceAccount(saName1)}
				c.ClusterRoleBindings = []rbacv1.ClusterRoleBinding{newClusterRoleBinding("cluster-role-binding", newClusterRoleRef(cRoleName1), newServiceAccountSubject(saName1))}
				applyClusterRoles(c, nil, strategy, nil)
				Expect(strategy.ClusterPermissions).To(Equal([]operatorsv1alpha1.StrategyDeploymentPermissions{}))
			})
		})
	})
})

var _ = Describe("applyCustomResourceDefinitions", func() {
	var c *collector.Manifests

	csv := operatorsv1alpha1.ClusterServiceVersion{
		Spec: operatorsv1alpha1.ClusterServiceVersionSpec{
			CustomResourceDefinitions: operatorsv1alpha1.CustomResourceDefinitions{
				Owned: []operatorsv1alpha1.CRDDescription{
					{
						Name:    "test1",
						Version: "v1",
						Kind:    "Memcached",
					},
					{
						Name:    "test1",
						Version: "v1beta1",
						Kind:    "Memcached",
					},
				},
			},
		},
	}

	It("should add all CRDs present in collector and specified in CSV", func() {
		c = &collector.Manifests{}
		crd1 := apiextv1.CustomResourceDefinition{
			Spec: apiextv1.CustomResourceDefinitionSpec{
				Group: "Test",
				Names: apiextv1.CustomResourceDefinitionNames{
					Kind: "Memcached",
				},
				Versions: []apiextv1.CustomResourceDefinitionVersion{
					{
						Name:   "v1",
						Served: true,
					},
					{
						Name:   "v1beta1",
						Served: true,
					},
				},
			}}
		crd1.SetName("test1")

		c.V1CustomResourceDefinitions = []apiextv1.CustomResourceDefinition{crd1}

		applyCustomResourceDefinitions(c, &csv)

		By("test if csv has the required owned crds applied")
		ownedDes := csv.Spec.CustomResourceDefinitions.Owned
		Expect(len(ownedDes)).To(BeEquivalentTo(2))
		Expect(ownedDes).To(ContainElements(operatorsv1alpha1.CRDDescription{
			Name:    "test1",
			Version: "v1",
			Kind:    "Memcached",
		}, operatorsv1alpha1.CRDDescription{
			Name:    "test1",
			Version: "v1beta1",
			Kind:    "Memcached",
		}))
	})

	It("should not add unserved v1CRDs", func() {
		c = &collector.Manifests{}
		crd1 := apiextv1.CustomResourceDefinition{
			Spec: apiextv1.CustomResourceDefinitionSpec{
				Group: "Test",
				Names: apiextv1.CustomResourceDefinitionNames{
					Kind: "Memcached",
				},
				Versions: []apiextv1.CustomResourceDefinitionVersion{
					{
						Name:   "v1",
						Served: true,
					},
					{
						Name:   "v1beta1",
						Served: false,
					},
				},
			}}
		crd1.SetName("test1")

		c.V1CustomResourceDefinitions = []apiextv1.CustomResourceDefinition{crd1}

		applyCustomResourceDefinitions(c, &csv)

		By("test if deprecated crds are not added")
		ownedDes := csv.Spec.CustomResourceDefinitions.Owned
		Expect(len(ownedDes)).To(BeEquivalentTo(1))
		Expect(ownedDes).To(ContainElement(operatorsv1alpha1.CRDDescription{
			Name:    "test1",
			Version: "v1",
			Kind:    "Memcached",
		}))
	})

	It("should not add unserved v1beta1CRDs", func() {
		c = &collector.Manifests{}

		crd1 := apiextv1beta1.CustomResourceDefinition{
			Spec: apiextv1beta1.CustomResourceDefinitionSpec{
				Group: "Test",
				Names: apiextv1beta1.CustomResourceDefinitionNames{
					Kind: "Memcached",
				},
				Versions: []apiextv1beta1.CustomResourceDefinitionVersion{
					{
						Name:   "v2",
						Served: true,
					},
					{
						Name:   "v1beta1",
						Served: false,
					},
				},
			},
		}

		crd1.SetName("test1")

		c.V1beta1CustomResourceDefinitions = []apiextv1beta1.CustomResourceDefinition{crd1}

		applyCustomResourceDefinitions(c, &csv)

		By("test if deprecated crds are not added")
		ownedDes := csv.Spec.CustomResourceDefinitions.Owned
		Expect(len(ownedDes)).To(BeEquivalentTo(1))
		Expect(ownedDes).To(ContainElement(operatorsv1alpha1.CRDDescription{
			Name:    "test1",
			Version: "v2",
			Kind:    "Memcached",
		}))
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
		It("parsing one deployment and one service with one label, it will returns the first service and deployment", func() {
			labels := map[string]string{"operator-name": "test-operator"}
			c.Deployments = []appsv1.Deployment{newDeployment(depName1, labels)}
			c.Services = []corev1.Service{newService(serviceName1, labels)}
			wcc.Service.Name = serviceName1
			depName, service := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(Equal(depName1))
			Expect(service.GetName()).To(Equal(serviceName1))
		})

		It("parsing two deployments and two services with non-intersecting labels, it will returns the first service and deployment", func() {
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

		It("parsing two deployments and two services with a label subset, it will returns the first service and second deployment", func() {
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
		It("parsing one deployment and one service with one label, it will returns neither service nor deployment name", func() {
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
		It("parsing one deployment and one service with one label, it will returns the first service and no deployment", func() {
			labels1 := map[string]string{"operator-name": "test-operator"}
			labels2 := map[string]string{"foo": "bar"}
			c.Deployments = []appsv1.Deployment{newDeployment(depName1, labels1)}
			c.Services = []corev1.Service{newService(serviceName1, labels2)}
			wcc.Service.Name = serviceName1
			depName, service := findMatchingDeploymentAndServiceForWebhook(c, wcc)
			Expect(depName).To(BeEmpty())
			Expect(service.GetName()).To(Equal(serviceName1))
		})

		It("parsing one deployment and one service with two intersecting labels, it will returns the first service and no deployment", func() {
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

	Context("crdGroups", func() {
		path1 := "/whPath"
		port1 := new(int32)
		*port1 = 2311
		crdToConfigPath := map[string]apiextv1.WebhookConversion{
			"crd-test-1": {
				ClientConfig: &apiextv1.WebhookClientConfig{
					Service: &apiextv1.ServiceReference{
						Path: &path1,
						Port: port1,
					},
				},
			},

			"crd-test-2": {
				ClientConfig: &apiextv1.WebhookClientConfig{
					Service: &apiextv1.ServiceReference{
						Path: &path1,
						Port: port1,
					},
				},
			},
		}

		val := crdGroups(crdToConfigPath)

		Expect(len(val)).To(BeEquivalentTo(1))

		test := serviceportPath{
			Port: port1,
			Path: path1,
		}

		g := val[test]

		Expect(g).NotTo(BeNil())
		Expect(len(g)).To(BeEquivalentTo(2))
		Expect(g).To(ContainElement("crd-test-2"))
		Expect(g).To(ContainElement("crd-test-1"))

	})

})

func newDeployment(name string, podLabels map[string]string) (dep appsv1.Deployment) {
	dep.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	dep.SetName(name)
	dep.Spec.Template.SetLabels(podLabels)
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

// newDeploymentWithLabels returns a deployment with the given labels
func newDeploymentWithLabels(name string, labels labels.Set) appsv1.Deployment {
	d := newDeployment(name, nil)
	d.ObjectMeta.Labels = labels
	return d
}

func newRole(name string, rules ...rbacv1.PolicyRule) (r *rbacv1.Role) {
	r = &rbacv1.Role{}
	r.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("Role"))
	r.SetName(name)
	r.Rules = rules
	return r
}

func newClusterRole(name string, rules ...rbacv1.PolicyRule) (r *rbacv1.ClusterRole) {
	r = &rbacv1.ClusterRole{}
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
