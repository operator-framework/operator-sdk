package collector

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Manifests", func() {
	var m Manifests
	BeforeEach(func() {
		m = Manifests{}
	})

	Describe("addRoles", func() {
		It("should unset the namespace", func() {
			objs := [][]byte{
				[]byte(string(`{"apiVersion":"rbac.authorization.k8s.io/v1", "kind":"Role", "metadata":{"name": "foo", "namespace":"foo"}}`)),
				[]byte(string(`{"apiVersion":"rbac.authorization.k8s.io/v1", "kind":"Role", "metadata":{"name": "bar"}}`)),
			}
			Expect(m.addRoles(objs...)).To(Succeed())
			Expect(m.Roles).To(HaveLen(2))
			for _, obj := range m.Roles {
				Expect(obj.GetNamespace()).To(BeEmpty())
			}
		})
	})

	Describe("addRoleBindings", func() {
		It("should unset the namespace", func() {
			objs := [][]byte{
				[]byte(string(`{"apiVersion":"rbac.authorization.k8s.io/v1", "kind":"RoleBinding", "metadata":{"name": "foo", "namespace":"foo"}}`)),
				[]byte(string(`{"apiVersion":"rbac.authorization.k8s.io/v1", "kind":"RoleBinding", "metadata":{"name": "bar"}}`)),
			}
			Expect(m.addRoleBindings(objs...)).To(Succeed())
			Expect(m.RoleBindings).To(HaveLen(2))
			for _, obj := range m.RoleBindings {
				Expect(obj.GetNamespace()).To(BeEmpty())
			}
		})
	})

	Describe("addServiceAccounts", func() {
		It("should unset the namespace", func() {
			objs := [][]byte{
				[]byte(string(`{"apiVersion":"v1", "kind":"ServiceAccount", "metadata":{"name": "foo", "namespace":"foo"}}`)),
				[]byte(string(`{"apiVersion":"v1", "kind":"ServiceAccount", "metadata":{"name": "bar"}}`)),
			}
			Expect(m.addServiceAccounts(objs...)).To(Succeed())
			Expect(m.ServiceAccounts).To(HaveLen(2))
			for _, obj := range m.ServiceAccounts {
				Expect(obj.GetNamespace()).To(BeEmpty())
			}
		})
	})

	Describe("addDeployments", func() {
		It("should unset the namespace", func() {
			objs := [][]byte{
				[]byte(string(`{"apiVersion":"apps/v1", "kind":"Deployment", "metadata":{"name": "foo", "namespace":"foo"}}`)),
				[]byte(string(`{"apiVersion":"apps/v1", "kind":"Deployment", "metadata":{"name": "bar"}}`)),
			}
			Expect(m.addDeployments(objs...)).To(Succeed())
			Expect(m.Deployments).To(HaveLen(2))
			for _, obj := range m.Deployments {
				Expect(obj.GetNamespace()).To(BeEmpty())
			}
		})
	})

	Describe("addOthers", func() {
		It("should unset the namespace", func() {
			objs := [][]byte{
				[]byte(string(`{"apiVersion":"example.com/v1", "kind":"Custom", "metadata":{"name": "foo", "namespace":"foo"}}`)),
				[]byte(string(`{"apiVersion":"example.com/v1", "kind":"Custom", "metadata":{"name": "bar"}}`)),
			}
			Expect(m.addOthers(objs...)).To(Succeed())
			Expect(m.Others).To(HaveLen(2))
			for _, obj := range m.Others {
				Expect(obj.GetNamespace()).To(BeEmpty())
			}
		})
	})
})
