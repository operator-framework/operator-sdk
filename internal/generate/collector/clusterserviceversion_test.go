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

//nolint:dupl
package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("ClusterServiceVersion", func() {
	var (
		c       *Manifests
		in, out []controllerutil.Object
	)

	BeforeEach(func() {
		c = &Manifests{}
	})

	Describe("SplitCSVPermissionsObjects", func() {

		It("should return empty lists for an empty Manifests", func() {
			c.Roles = []rbacv1.Role{}
			in, out = c.SplitCSVPermissionsObjects()
			Expect(in).To(HaveLen(0))
			Expect(out).To(HaveLen(0))
		})
		It("should return non-empty lists", func() {
			By("splitting 1 Role no RoleBinding")
			c.Roles = []rbacv1.Role{newRole("my-role")}
			in, out = c.SplitCSVPermissionsObjects()
			Expect(in).To(HaveLen(0))
			Expect(out).To(HaveLen(1))
			Expect(getRoleNames(out)).To(ContainElement("my-role"))

			By("splitting 1 Role 1 RoleBinding with 1 Subject not containing Deployment serviceAccountName")
			c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
			c.Roles = []rbacv1.Role{newRole("my-role")}
			c.RoleBindings = []rbacv1.RoleBinding{
				newRoleBinding("my-role-binding", newRoleRef("my-role"), newServiceAccountSubject("my-other-account")),
			}
			in, out = c.SplitCSVPermissionsObjects()
			Expect(in).To(HaveLen(0))
			Expect(out).To(HaveLen(2))
			Expect(getRoleNames(out)).To(ContainElement("my-role"))
			Expect(getRoleBindingNames(out)).To(ContainElement("my-role-binding"))

			By("splitting 1 Role 1 RoleBinding with 1 Subject containing Deployment serviceAccountName")
			c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
			c.Roles = []rbacv1.Role{newRole("my-role")}
			c.RoleBindings = []rbacv1.RoleBinding{
				newRoleBinding("my-role-binding", newRoleRef("my-role"), newServiceAccountSubject("my-dep-account")),
			}
			in, out = c.SplitCSVPermissionsObjects()
			Expect(in).To(HaveLen(1))
			Expect(getRoleNames(in)).To(ContainElement("my-role"))
			Expect(out).To(HaveLen(0))

			By("splitting 1 Role 1 RoleBinding with 2 Subjects containing a Deployment serviceAccountName")
			c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
			c.Roles = []rbacv1.Role{newRole("my-role")}
			c.RoleBindings = []rbacv1.RoleBinding{
				newRoleBinding("my-role-binding",
					newRoleRef("my-role"),
					newServiceAccountSubject("my-dep-account"), newServiceAccountSubject("my-other-account")),
			}
			in, out = c.SplitCSVPermissionsObjects()
			Expect(in).To(HaveLen(1))
			Expect(getRoleNames(in)).To(ContainElement("my-role"))
			Expect(out).To(HaveLen(2))
			Expect(getRoleNames(out)).To(ContainElement("my-role"))
			Expect(getRoleBindingNames(out)).To(ContainElement("my-role-binding"))

			By("splitting 2 Roles 2 RoleBinding, one with 1 Subject not containing and the other with 2 Subjects containing a Deployment serviceAccountName")
			c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
			c.Roles = []rbacv1.Role{newRole("my-role-1"), newRole("my-role-2")}
			c.RoleBindings = []rbacv1.RoleBinding{
				newRoleBinding("my-role-binding-1",
					newRoleRef("my-role-1"),
					newServiceAccountSubject("my-dep-account"), newServiceAccountSubject("my-other-account")),
				newRoleBinding("my-role-binding-2",
					newRoleRef("my-role-2"),
					newServiceAccountSubject("my-other-account")),
			}
			in, out = c.SplitCSVPermissionsObjects()
			Expect(in).To(HaveLen(1))
			Expect(getRoleNames(in)).To(ContainElement("my-role-1"))
			Expect(out).To(HaveLen(4))
			Expect(getRoleNames(out)).To(ContainElement("my-role-1"))
			Expect(getRoleNames(out)).To(ContainElement("my-role-2"))
			Expect(getRoleBindingNames(out)).To(ContainElement("my-role-binding-1"))
			Expect(getRoleBindingNames(out)).To(ContainElement("my-role-binding-2"))

			By("splitting on 2 different Deployments")
			c.Deployments = []appsv1.Deployment{
				newDeploymentWithServiceAccount("my-dep-account-1"),
				newDeploymentWithServiceAccount("my-dep-account-2"),
			}
			c.Roles = []rbacv1.Role{newRole("my-role-1"), newRole("my-role-2"), newRole("my-role-3")}
			c.RoleBindings = []rbacv1.RoleBinding{
				newRoleBinding("my-role-binding-1",
					newRoleRef("my-role-1"),
					newServiceAccountSubject("my-dep-account-1"), newServiceAccountSubject("my-other-account")),
				newRoleBinding("my-role-binding-2",
					newRoleRef("my-role-2"),
					newServiceAccountSubject("my-other-account")),
				newRoleBinding("my-role-binding-3",
					newRoleRef("my-role-3"),
					newServiceAccountSubject("my-dep-account-2")),
			}
			in, out = c.SplitCSVPermissionsObjects()
			Expect(in).To(HaveLen(2))
			Expect(getRoleNames(in)).To(ContainElement("my-role-1"))
			Expect(getRoleNames(in)).To(ContainElement("my-role-3"))
			Expect(out).To(HaveLen(4))
			Expect(getRoleNames(out)).To(ContainElement("my-role-1"))
			Expect(getRoleNames(out)).To(ContainElement("my-role-2"))
			Expect(getRoleBindingNames(out)).To(ContainElement("my-role-binding-1"))
			Expect(getRoleBindingNames(out)).To(ContainElement("my-role-binding-2"))
		})
	})

	Describe("SplitCSVClusterPermissionsObjects", func() {
		It("should return empty lists for an empty Manifests", func() {
			c.ClusterRoles = []rbacv1.ClusterRole{}
			in, out = c.SplitCSVClusterPermissionsObjects()
			Expect(in).To(HaveLen(0))
			Expect(out).To(HaveLen(0))
		})
		It("should return non-empty lists", func() {
			By("splitting 1 ClusterRole no ClusterRoleBinding")
			c.ClusterRoles = []rbacv1.ClusterRole{newClusterRole("my-role")}
			in, out = c.SplitCSVClusterPermissionsObjects()
			Expect(in).To(HaveLen(0))
			Expect(out).To(HaveLen(1))
			Expect(getClusterRoleNames(out)).To(ContainElement("my-role"))

			By("splitting 1 ClusterRole 1 ClusterRoleBinding with 1 Subject not containing Deployment serviceAccountName")
			c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
			c.ClusterRoles = []rbacv1.ClusterRole{newClusterRole("my-role")}
			c.ClusterRoleBindings = []rbacv1.ClusterRoleBinding{
				newClusterRoleBinding("my-role-binding", newClusterRoleRef("my-role"), newServiceAccountSubject("my-other-account")),
			}
			in, out = c.SplitCSVClusterPermissionsObjects()
			Expect(in).To(HaveLen(0))
			Expect(out).To(HaveLen(2))
			Expect(getClusterRoleNames(out)).To(ContainElement("my-role"))
			Expect(getClusterRoleBindingNames(out)).To(ContainElement("my-role-binding"))

			By("splitting 1 ClusterRole 1 ClusterRoleBinding with 1 Subject containing Deployment serviceAccountName")
			c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
			c.ClusterRoles = []rbacv1.ClusterRole{newClusterRole("my-role")}
			c.ClusterRoleBindings = []rbacv1.ClusterRoleBinding{
				newClusterRoleBinding("my-role-binding", newClusterRoleRef("my-role"), newServiceAccountSubject("my-dep-account")),
			}
			in, out = c.SplitCSVClusterPermissionsObjects()
			Expect(in).To(HaveLen(1))
			Expect(getClusterRoleNames(in)).To(ContainElement("my-role"))
			Expect(out).To(HaveLen(0))

			By("splitting 1 ClusterRole 1 ClusterRoleBinding with 2 Subjects containing a Deployment serviceAccountName")
			c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
			c.ClusterRoles = []rbacv1.ClusterRole{newClusterRole("my-role")}
			c.ClusterRoleBindings = []rbacv1.ClusterRoleBinding{
				newClusterRoleBinding("my-role-binding",
					newClusterRoleRef("my-role"),
					newServiceAccountSubject("my-dep-account"), newServiceAccountSubject("my-other-account")),
			}
			in, out = c.SplitCSVClusterPermissionsObjects()
			Expect(in).To(HaveLen(1))
			Expect(getClusterRoleNames(in)).To(ContainElement("my-role"))
			Expect(out).To(HaveLen(2))
			Expect(getClusterRoleNames(out)).To(ContainElement("my-role"))
			Expect(getClusterRoleBindingNames(out)).To(ContainElement("my-role-binding"))

			By("splitting 2 ClusterRoles 2 ClusterRoleBindings, one with 1 Subject not containing and the other with 2 Subjects containing a Deployment serviceAccountName")
			c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
			c.ClusterRoles = []rbacv1.ClusterRole{newClusterRole("my-role-1"), newClusterRole("my-role-2")}
			c.ClusterRoleBindings = []rbacv1.ClusterRoleBinding{
				newClusterRoleBinding("my-role-binding-1",
					newClusterRoleRef("my-role-1"),
					newServiceAccountSubject("my-dep-account"), newServiceAccountSubject("my-other-account")),
				newClusterRoleBinding("my-role-binding-2",
					newClusterRoleRef("my-role-2"),
					newServiceAccountSubject("my-other-account")),
			}
			in, out = c.SplitCSVClusterPermissionsObjects()
			Expect(in).To(HaveLen(1))
			Expect(getClusterRoleNames(in)).To(ContainElement("my-role-1"))
			Expect(out).To(HaveLen(4))
			Expect(getClusterRoleNames(out)).To(ContainElement("my-role-1"))
			Expect(getClusterRoleNames(out)).To(ContainElement("my-role-2"))
			Expect(getClusterRoleBindingNames(out)).To(ContainElement("my-role-binding-1"))
			Expect(getClusterRoleBindingNames(out)).To(ContainElement("my-role-binding-2"))

			By("splitting on 2 different Deployments")
			c.Deployments = []appsv1.Deployment{
				newDeploymentWithServiceAccount("my-dep-account-1"),
				newDeploymentWithServiceAccount("my-dep-account-2"),
			}
			c.ClusterRoles = []rbacv1.ClusterRole{
				newClusterRole("my-role-1"),
				newClusterRole("my-role-2"),
				newClusterRole("my-role-3"),
			}
			c.ClusterRoleBindings = []rbacv1.ClusterRoleBinding{
				newClusterRoleBinding("my-role-binding-1",
					newClusterRoleRef("my-role-1"),
					newServiceAccountSubject("my-dep-account-1"), newServiceAccountSubject("my-other-account")),
				newClusterRoleBinding("my-role-binding-2",
					newClusterRoleRef("my-role-2"),
					newServiceAccountSubject("my-other-account")),
				newClusterRoleBinding("my-role-binding-3",
					newClusterRoleRef("my-role-3"),
					newServiceAccountSubject("my-dep-account-2")),
			}
			in, out = c.SplitCSVClusterPermissionsObjects()
			Expect(in).To(HaveLen(2))
			Expect(getClusterRoleNames(in)).To(ContainElement("my-role-1"))
			Expect(getClusterRoleNames(in)).To(ContainElement("my-role-3"))
			Expect(out).To(HaveLen(4))
			Expect(getClusterRoleNames(out)).To(ContainElement("my-role-1"))
			Expect(getClusterRoleNames(out)).To(ContainElement("my-role-2"))
			Expect(getClusterRoleBindingNames(out)).To(ContainElement("my-role-binding-1"))
			Expect(getClusterRoleBindingNames(out)).To(ContainElement("my-role-binding-2"))
		})
	})

})

func getRoleNames(objs []controllerutil.Object) []string {
	return getNamesForKind("Role", objs)
}

func getRoleBindingNames(objs []controllerutil.Object) []string {
	return getNamesForKind("RoleBinding", objs)
}

func getClusterRoleNames(objs []controllerutil.Object) []string {
	return getNamesForKind("ClusterRole", objs)
}

func getClusterRoleBindingNames(objs []controllerutil.Object) []string {
	return getNamesForKind("ClusterRoleBinding", objs)
}

func getNamesForKind(kind string, objs []controllerutil.Object) (names []string) {
	for _, obj := range objs {
		if obj.GetObjectKind().GroupVersionKind().Kind == kind {
			names = append(names, obj.GetName())
		}
	}
	return
}

func newDeploymentWithServiceAccount(name string) (d appsv1.Deployment) {
	d.Spec.Template.Spec.ServiceAccountName = name
	return d
}

func newRole(name string) (r rbacv1.Role) {
	r.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("Role"))
	r.SetName(name)
	return r
}

func newClusterRole(name string) (r rbacv1.ClusterRole) {
	r.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("ClusterRole"))
	r.SetName(name)
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

func newServiceAccountSubject(name string) (s rbacv1.Subject) {
	return newSubject(name, "ServiceAccount")
}
