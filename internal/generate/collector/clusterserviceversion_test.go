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
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("SplitCSVPermissionsObjects", func() {
	var c *Manifests
	var inPerm, inCPerm, out []client.Object

	BeforeEach(func() {
		c = &Manifests{}
	})

	It("returns empty lists for an empty Manifests", func() {
		c.Roles = []rbacv1.Role{}
		inPerm, inCPerm, out = c.SplitCSVPermissionsObjects(nil)
		Expect(inPerm).To(BeEmpty())
		Expect(inCPerm).To(BeEmpty())
		Expect(out).To(BeEmpty())
	})

	It("splitting 1 Role no RoleBinding", func() {
		c.Roles = []rbacv1.Role{newRole("my-role")}
		inPerm, inCPerm, out = c.SplitCSVPermissionsObjects(nil)
		Expect(inPerm).To(BeEmpty())
		Expect(inCPerm).To(BeEmpty())
		Expect(out).To(HaveLen(1))
		Expect(getRoleNames(out)).To(Equal([]string{"my-role"}))
	})

	It("splitting 1 Role 1 RoleBinding with 1 Subject not containing Deployment serviceAccountName", func() {
		c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
		c.Roles = []rbacv1.Role{newRole("my-role")}
		c.RoleBindings = []rbacv1.RoleBinding{
			newRoleBinding("my-role-binding", newRoleRef("my-role"), newServiceAccountSubject("my-other-account")),
		}
		inPerm, inCPerm, out = c.SplitCSVPermissionsObjects(nil)
		Expect(inPerm).To(BeEmpty())
		Expect(inCPerm).To(BeEmpty())
		Expect(out).To(HaveLen(2))
		Expect(getRoleNames(out)).To(Equal([]string{"my-role"}))
		Expect(getRoleBindingNames(out)).To(Equal([]string{"my-role-binding"}))
	})

	It("splitting 1 ClusterRole 1 RoleBinding with 1 Subject not containing Deployment serviceAccountName", func() {
		c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
		c.ClusterRoles = []rbacv1.ClusterRole{newClusterRole("my-role")}
		c.RoleBindings = []rbacv1.RoleBinding{
			newRoleBinding("my-role-binding", newClusterRoleRef("my-role"), newServiceAccountSubject("my-other-account")),
		}
		inPerm, inCPerm, out = c.SplitCSVPermissionsObjects(nil)
		Expect(inPerm).To(BeEmpty())
		Expect(inCPerm).To(BeEmpty())
		Expect(out).To(HaveLen(2))
		Expect(getClusterRoleNames(out)).To(Equal([]string{"my-role"}))
		Expect(getRoleBindingNames(out)).To(Equal([]string{"my-role-binding"}))
	})

	It("splitting 1 Role 1 ClusterRole 1 RoleBinding with 1 Subject not containing Deployment serviceAccountName", func() {
		c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
		c.Roles = []rbacv1.Role{newRole("my-role")}
		c.ClusterRoles = []rbacv1.ClusterRole{newClusterRole("my-role")}
		c.RoleBindings = []rbacv1.RoleBinding{
			newRoleBinding("my-role-binding-1", newRoleRef("my-role"), newServiceAccountSubject("my-other-account")),
			newRoleBinding("my-role-binding-2", newClusterRoleRef("my-role"), newServiceAccountSubject("my-other-account")),
		}
		inPerm, inCPerm, out = c.SplitCSVPermissionsObjects(nil)
		Expect(inPerm).To(BeEmpty())
		Expect(inCPerm).To(BeEmpty())
		Expect(out).To(HaveLen(4))
		Expect(getRoleNames(out)).To(Equal([]string{"my-role"}))
		Expect(getClusterRoleNames(out)).To(Equal([]string{"my-role"}))
		Expect(getRoleBindingNames(out)).To(Equal([]string{"my-role-binding-1", "my-role-binding-2"}))
	})

	It("splitting 1 Role 1 RoleBinding with 1 Subject containing a Deployment serviceAccountName", func() {
		c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
		c.Roles = []rbacv1.Role{newRole("my-role")}
		c.RoleBindings = []rbacv1.RoleBinding{
			newRoleBinding("my-role-binding", newRoleRef("my-role"), newServiceAccountSubject("my-dep-account")),
		}
		inPerm, inCPerm, out = c.SplitCSVPermissionsObjects(nil)
		Expect(inPerm).To(HaveLen(1))
		Expect(getRoleNames(inPerm)).To(Equal([]string{"my-role"}))
		Expect(inCPerm).To(BeEmpty())
		Expect(out).To(BeEmpty())
	})

	It("splitting 1 ClusterRole 1 RoleBinding with 1 Subject containing a Deployment serviceAccountName", func() {
		c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
		c.ClusterRoles = []rbacv1.ClusterRole{newClusterRole("my-role")}
		c.RoleBindings = []rbacv1.RoleBinding{
			newRoleBinding("my-role-binding", newClusterRoleRef("my-role"), newServiceAccountSubject("my-dep-account")),
		}
		inPerm, inCPerm, out = c.SplitCSVPermissionsObjects(nil)
		Expect(inPerm).To(HaveLen(1))
		Expect(getClusterRoleNames(inPerm)).To(Equal([]string{"my-role"}))
		Expect(inCPerm).To(BeEmpty())
		Expect(out).To(BeEmpty())
	})

	It("splitting 1 Role 1 ClusterRole 1 RoleBinding with 1 Subject containing a Deployment serviceAccountName", func() {
		c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
		c.Roles = []rbacv1.Role{newRole("my-role")}
		c.ClusterRoles = []rbacv1.ClusterRole{newClusterRole("my-role")}
		c.RoleBindings = []rbacv1.RoleBinding{
			newRoleBinding("my-role-binding-1", newRoleRef("my-role"), newServiceAccountSubject("my-dep-account")),
			newRoleBinding("my-role-binding-2", newClusterRoleRef("my-role"), newServiceAccountSubject("my-dep-account")),
		}
		inPerm, inCPerm, out = c.SplitCSVPermissionsObjects(nil)
		Expect(inPerm).To(HaveLen(2))
		Expect(getRoleNames(inPerm)).To(Equal([]string{"my-role"}))
		Expect(getClusterRoleNames(inPerm)).To(Equal([]string{"my-role"}))
		Expect(inCPerm).To(BeEmpty())
		Expect(out).To(BeEmpty())
	})

	It("splitting 1 Role 1 ClusterRole 1 RoleBinding with 2 Subjects containing a Deployment serviceAccountName", func() {
		c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount("my-dep-account")}
		c.Roles = []rbacv1.Role{newRole("my-role")}
		c.ClusterRoles = []rbacv1.ClusterRole{newClusterRole("my-role")}
		c.RoleBindings = []rbacv1.RoleBinding{
			newRoleBinding("my-role-binding-1", newRoleRef("my-role"), newServiceAccountSubject("my-other-account"), newServiceAccountSubject("my-dep-account")),
			newRoleBinding("my-role-binding-2", newClusterRoleRef("my-role"), newServiceAccountSubject("my-dep-account")),
		}
		inPerm, inCPerm, out = c.SplitCSVPermissionsObjects(nil)
		Expect(inPerm).To(HaveLen(2))
		Expect(getRoleNames(inPerm)).To(Equal([]string{"my-role"}))
		Expect(getClusterRoleNames(inPerm)).To(Equal([]string{"my-role"}))
		Expect(out).To(HaveLen(1))
		Expect(getRoleBindingNames(out)).To(Equal([]string{"my-role-binding-1"}))
	})

	Context("multiple relationship RBAC", func() {
		depSA, extraSA := "my-dep-account", "my-other-account"
		roleName1, roleName2 := "my-role-1", "my-role-2"
		bindingName1, bindingName2, bindingName3 := "my-role-binding-1", "my-role-binding-2", "my-role-binding-3"

		complexTestSetup := func() {
			c.Deployments = []appsv1.Deployment{newDeploymentWithServiceAccount(depSA)}
			c.ServiceAccounts = []corev1.ServiceAccount{
				newServiceAccount(depSA),
				newServiceAccount(extraSA),
			}
			role1, role2 := newRole(roleName1), newRole(roleName2)
			c.Roles = []rbacv1.Role{role1, role2}
			// Use the same names as for Roles to make sure Kind is respected
			cRole1, cRole2 := newClusterRole(roleName1), newClusterRole(roleName2)
			c.ClusterRoles = []rbacv1.ClusterRole{cRole1, cRole2}

			// Binds role 1 to depSA,extraSA, role 2 to extraSA, and clusterrole 1 to depSA.
			c.RoleBindings = []rbacv1.RoleBinding{
				newRoleBinding(bindingName1,
					newRoleRef(role1.Name),
					newServiceAccountSubject(depSA), newServiceAccountSubject(extraSA)),
				newRoleBinding(bindingName2,
					newRoleRef(role2.Name),
					newServiceAccountSubject(extraSA)),
				newRoleBinding(bindingName3,
					newClusterRoleRef(cRole1.Name),
					newServiceAccountSubject(depSA)),
			}

			// Binds clusterrole 1 to depSA and clusterrole 2 to extraSA.
			c.ClusterRoleBindings = []rbacv1.ClusterRoleBinding{
				newClusterRoleBinding(bindingName1,
					newClusterRoleRef(cRole1.Name),
					newServiceAccountSubject(depSA)),
				newClusterRoleBinding(bindingName2,
					newClusterRoleRef(cRole2.Name),
					newServiceAccountSubject(extraSA)),
			}
		}

		It("contains a Deployment serviceAccountName only", func() {
			complexTestSetup()
			inPerm, inCPerm, out = c.SplitCSVPermissionsObjects(nil)
			Expect(inPerm).To(HaveLen(2))
			Expect(getRoleNames(inPerm)).To(Equal([]string{roleName1}))
			Expect(getClusterRoleNames(inPerm)).To(Equal([]string{roleName1}))
			Expect(inCPerm).To(HaveLen(1))
			Expect(getClusterRoleNames(inCPerm)).To(Equal([]string{roleName1}))
			Expect(out).To(HaveLen(6))
			Expect(getRoleNames(out)).To(Equal([]string{roleName2}))
			Expect(getClusterRoleNames(out)).To(Equal([]string{roleName2}))
			Expect(getRoleBindingNames(out)).To(Equal([]string{bindingName1, bindingName2}))
			Expect(getClusterRoleBindingNames(out)).To(Equal([]string{bindingName2}))
			Expect(getServiceAccountNames(out)).To(Equal([]string{extraSA}))
		})

		It("contains a Deployment serviceAccountName and extra ServiceAccount", func() {
			complexTestSetup()
			inPerm, inCPerm, out = c.SplitCSVPermissionsObjects([]string{extraSA})
			Expect(inPerm).To(HaveLen(3))
			Expect(getRoleNames(inPerm)).To(Equal([]string{roleName1, roleName2}))
			Expect(getClusterRoleNames(inPerm)).To(Equal([]string{roleName1}))
			Expect(inCPerm).To(HaveLen(2))
			Expect(getClusterRoleNames(inCPerm)).To(Equal([]string{roleName1, roleName2}))
			Expect(out).To(BeEmpty())
		})
	})
})

func getRoleNames(objs []client.Object) []string {
	return getNamesForKind("Role", objs)
}

func getRoleBindingNames(objs []client.Object) []string {
	return getNamesForKind("RoleBinding", objs)
}

func getClusterRoleNames(objs []client.Object) []string {
	return getNamesForKind("ClusterRole", objs)
}

func getClusterRoleBindingNames(objs []client.Object) []string {
	return getNamesForKind("ClusterRoleBinding", objs)
}

func getServiceAccountNames(objs []client.Object) []string {
	return getNamesForKind("ServiceAccount", objs)
}

func getNamesForKind(kind string, objs []client.Object) (names []string) {
	for _, obj := range objs {
		if obj.GetObjectKind().GroupVersionKind().Kind == kind {
			names = append(names, obj.GetName())
		}
	}
	sort.Strings(names)
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

func newServiceAccount(name string) (sa corev1.ServiceAccount) {
	sa.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ServiceAccount"))
	sa.Name = name
	return sa
}
