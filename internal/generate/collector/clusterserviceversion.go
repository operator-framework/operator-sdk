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

package collector

import (
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO(estroz): there's a significant amount of code dupliation here, a byproduct of Go's type system.
// However at least a few bits can be refactored so each method is smaller.

const (
	// This service account exists in every namespace as the default.
	defaultServiceAccountName = "default"

	serviceAccountKind = "ServiceAccount"
	roleKind           = "Role"
	cRoleKind          = "ClusterRole"
)

// SplitCSVPermissionsObjects splits Roles and ClusterRoles bound to ServiceAccounts in Deployments and extraSA's.
// Roles and ClusterRoles bound to RoleBindings associated with ServiceAccounts are added to inPerms.
// ClusterRoles bound to ClusterRoleBindings associated with ServiceAccounts are added to inCPerms.
// All unassociated Roles, ClusterRoles, and bindings are added to out.
// Any bindings with some associations, but with non-associations, are added to out unmodified.
func (c *Manifests) SplitCSVPermissionsObjects(extraSAs []string) (inPerms, inCPerms, out []client.Object) { //nolint:gocyclo
	// Create a set of ServiceAccount names to match against below.
	saNameSet := make(map[string]struct{})
	for _, dep := range c.Deployments {
		saName := dep.Spec.Template.Spec.ServiceAccountName
		if saName == "" {
			saName = defaultServiceAccountName
		}
		saNameSet[saName] = struct{}{}
	}
	for _, extraSA := range extraSAs {
		saNameSet[extraSA] = struct{}{}
	}

	roleBindings := copyRoleBindings(c.RoleBindings)
	cRoleBindings := copyClusterRoleBindings(c.ClusterRoleBindings)

	var inPermsR, inPermsCR, inCPermsCR []client.Object
	inPermsR, roleBindings = getRolesBoundToRoleBindings(c.Roles, roleBindings, saNameSet)
	inPerms = append(inPerms, inPermsR...)
	inPermsCR, roleBindings = getClusterRolesBoundToRoleBindings(c.ClusterRoles, roleBindings, saNameSet)
	inPerms = append(inPerms, inPermsCR...)
	inCPermsCR, cRoleBindings = getClusterRolesBoundToClusterRoleBindings(c.ClusterRoles, cRoleBindings, saNameSet)
	inCPerms = append(inCPerms, inCPermsCR...)

	roleMap := make(map[string]*rbacv1.Role, len(c.Roles))
	cRoleMap := make(map[string]*rbacv1.ClusterRole, len(c.ClusterRoles))
	for i, perm := range inPerms {
		switch t := perm.(type) {
		case *rbacv1.Role:
			roleMap[t.GetName()] = inPerms[i].(*rbacv1.Role)
		case *rbacv1.ClusterRole:
			cRoleMap[t.GetName()] = inPerms[i].(*rbacv1.ClusterRole)
		}
	}
	for i, perm := range inCPerms {
		if t, ok := perm.(*rbacv1.ClusterRole); ok {
			cRoleMap[t.GetName()] = inCPerms[i].(*rbacv1.ClusterRole)
		}
	}
	for i := 0; i < len(c.Roles); i++ {
		role := c.Roles[i]
		if _, inPerms := roleMap[role.GetName()]; !inPerms {
			out = append(out, &role)
		}
	}
	for i := 0; i < len(c.ClusterRoles); i++ {
		role := c.ClusterRoles[i]
		if _, inPerms := cRoleMap[role.GetName()]; !inPerms {
			out = append(out, &role)
		}
	}

	roleBindingMap := make(map[string]struct{}, len(roleBindings))
	cRoleBindingMap := make(map[string]struct{}, len(cRoleBindings))
	for _, binding := range roleBindings {
		roleBindingMap[binding.GetName()] = struct{}{}
	}
	for _, binding := range cRoleBindings {
		cRoleBindingMap[binding.GetName()] = struct{}{}
	}
	for i := range c.RoleBindings {
		binding := c.RoleBindings[i]
		if _, ok := roleBindingMap[binding.GetName()]; ok {
			out = append(out, &binding)
		}
	}
	for i := range c.ClusterRoleBindings {
		binding := c.ClusterRoleBindings[i]
		if _, ok := cRoleBindingMap[binding.GetName()]; ok {
			out = append(out, &binding)
		}
	}

	return inPerms, inCPerms, out
}

func copyRoleBindings(in []rbacv1.RoleBinding) (out []*rbacv1.RoleBinding) {
	out = make([]*rbacv1.RoleBinding, len(in))
	for i, binding := range in {
		out[i] = binding.DeepCopy()
	}
	return out
}

func copyClusterRoleBindings(in []rbacv1.ClusterRoleBinding) (out []*rbacv1.ClusterRoleBinding) {
	out = make([]*rbacv1.ClusterRoleBinding, len(in))
	for i, binding := range in {
		out[i] = binding.DeepCopy()
	}
	return out
}

// getRolesBoundToRoleBindings splits roles that should be written to a CSV as permissions (in)
// from roles and role bindings that should be written directly to the bundle (out).
//nolint:dupl
func getRolesBoundToRoleBindings(roles []rbacv1.Role, bindings []*rbacv1.RoleBinding, saNames map[string]struct{}) (in []client.Object, _ []*rbacv1.RoleBinding) {
	roleMap := make(map[string]*rbacv1.Role, len(roles))
	for i, r := range roles {
		roleMap[r.GetName()] = &roles[i]
	}

	for i := 0; i < len(bindings); i++ {
		binding := bindings[i]
		ref := binding.RoleRef
		role, hasRoleName := roleMap[ref.Name]
		if !hasRoleName || ref.Kind != roleKind || !acceptRefGroup(ref.APIGroup) {
			continue
		}
		addRole := false
		for j := 0; j < len(binding.Subjects); j++ {
			subject := binding.Subjects[j]
			if _, hasSA := saNames[subject.Name]; hasSA && subject.Kind == serviceAccountKind {
				addRole = true
				binding.Subjects = append(binding.Subjects[:j], binding.Subjects[j+1:]...)
				j--
			}
		}
		// At least one ServiceAccount of this binding in saNames was found, so add the role.
		if addRole {
			in = append(in, role)
		}
		if len(binding.Subjects) == 0 {
			bindings = append(bindings[:i], bindings[i+1:]...)
			i--
		}
	}

	return in, bindings
}

//nolint:dupl
func getClusterRolesBoundToRoleBindings(roles []rbacv1.ClusterRole, bindings []*rbacv1.RoleBinding, saNames map[string]struct{}) (in []client.Object, _ []*rbacv1.RoleBinding) {
	roleMap := make(map[string]*rbacv1.ClusterRole)
	for i, r := range roles {
		roleMap[r.GetName()] = &roles[i]
	}

	for i := 0; i < len(bindings); i++ {
		binding := bindings[i]
		ref := binding.RoleRef
		role, hasRoleName := roleMap[ref.Name]
		if !hasRoleName || ref.Kind != cRoleKind || !acceptRefGroup(ref.APIGroup) {
			continue
		}
		addRole := false
		for j := 0; j < len(binding.Subjects); j++ {
			subject := binding.Subjects[j]
			if _, hasSA := saNames[subject.Name]; hasSA && subject.Kind == serviceAccountKind {
				addRole = true
				binding.Subjects = append(binding.Subjects[:j], binding.Subjects[j+1:]...)
				j--
			}
		}
		// At least one ServiceAccount of this binding in saNames was found, so add the role.
		if addRole {
			in = append(in, role)
		}
		if len(binding.Subjects) == 0 {
			bindings = append(bindings[:i], bindings[i+1:]...)
			i--
		}
	}

	return in, bindings
}

//nolint:dupl
func getClusterRolesBoundToClusterRoleBindings(roles []rbacv1.ClusterRole, bindings []*rbacv1.ClusterRoleBinding, saNames map[string]struct{}) (in []client.Object, _ []*rbacv1.ClusterRoleBinding) {
	roleMap := make(map[string]*rbacv1.ClusterRole)
	for i, r := range roles {
		roleMap[r.GetName()] = &roles[i]
	}

	for i := 0; i < len(bindings); i++ {
		binding := bindings[i]
		ref := binding.RoleRef
		role, hasRoleName := roleMap[ref.Name]
		if !hasRoleName || ref.Kind != cRoleKind || !acceptRefGroup(ref.APIGroup) {
			continue
		}
		addRole := false
		for j := 0; j < len(binding.Subjects); j++ {
			subject := binding.Subjects[j]
			if _, hasSA := saNames[subject.Name]; hasSA && subject.Kind == serviceAccountKind {
				addRole = true
				binding.Subjects = append(binding.Subjects[:j], binding.Subjects[j+1:]...)
				j--
			}
		}
		// At least one ServiceAccount of this binding in saNames was found, so add the role.
		if addRole {
			in = append(in, role)
		}
		if len(binding.Subjects) == 0 {
			bindings = append(bindings[:i], bindings[i+1:]...)
			i--
		}
	}

	return in, bindings
}

func acceptRefGroup(apiGroup string) bool {
	return apiGroup == "" || apiGroup == rbacv1.SchemeGroupVersion.Group
}
