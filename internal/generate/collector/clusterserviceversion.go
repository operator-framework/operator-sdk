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
// All Roles and ClusterRoles not bound to a relevant ServiceAccount, and bindings without a relevant ServiceAccount,
// are added to out. Any bindings with some associations, but with non-associations, are added to out unmodified.
func (c *Manifests) SplitCSVPermissionsObjects(extraSAs []string) (inPerms, inCPerms, out []client.Object) {
	// Create a set of ServiceAccount names to match against below.
	csvSAs := make(map[string]struct{})
	for _, dep := range c.Deployments {
		saName := dep.Spec.Template.Spec.ServiceAccountName
		if saName == "" {
			saName = defaultServiceAccountName
		}
		csvSAs[saName] = struct{}{}
	}
	for _, extraSA := range extraSAs {
		csvSAs[extraSA] = struct{}{}
	}

	// Construct sets for lookups.
	roleMap := make(map[string]*rbacv1.Role, len(c.Roles))
	roleNameMap := make(map[string]struct{}, len(c.Roles))
	for i, r := range c.Roles {
		roleMap[r.GetName()] = &c.Roles[i]
		roleNameMap[r.GetName()] = struct{}{}
	}
	cRoleMap := make(map[string]*rbacv1.ClusterRole, len(c.ClusterRoles))
	cRoleNameMap := make(map[string]struct{}, len(c.ClusterRoles))
	for i, r := range c.ClusterRoles {
		cRoleMap[r.GetName()] = &c.ClusterRoles[i]
		cRoleNameMap[r.GetName()] = struct{}{}
	}
	pRoleBindings := make(partialBindings, len(c.RoleBindings))
	roleBindingMap := make(map[string]*rbacv1.RoleBinding, len(c.RoleBindings))
	for i, binding := range c.RoleBindings {
		pRoleBindings[i].Name = binding.GetName()
		pRoleBindings[i].RoleRef = binding.RoleRef
		pRoleBindings[i].Subjects = make([]rbacv1.Subject, len(binding.Subjects))
		copy(pRoleBindings[i].Subjects, binding.Subjects)
		roleBindingMap[binding.GetName()] = &c.RoleBindings[i]
	}
	pCRoleBindings := make(partialBindings, len(c.ClusterRoleBindings))
	cRoleBindingMap := make(map[string]*rbacv1.ClusterRoleBinding, len(c.ClusterRoleBindings))
	for i, binding := range c.ClusterRoleBindings {
		pCRoleBindings[i].Name = binding.GetName()
		pCRoleBindings[i].RoleRef = binding.RoleRef
		pCRoleBindings[i].Subjects = make([]rbacv1.Subject, len(binding.Subjects))
		copy(pCRoleBindings[i].Subjects, binding.Subjects)
		cRoleBindingMap[binding.GetName()] = &c.ClusterRoleBindings[i]
	}

	// getRolesBoundToPartialBindings will remove
	// bound Subjects from partial bindings to easily find concrete bindings that bind non-CSV RBAC.
	// Those with no Subjects left will be removed from the partial binding lists; their concrete counterparts should
	// be added to out.
	inRoleNames := pRoleBindings.getRolesBoundToPartialBindings(roleKind, roleNameMap, csvSAs)
	inCRoleNamesNScope := pRoleBindings.getRolesBoundToPartialBindings(cRoleKind, cRoleNameMap, csvSAs)
	inCRoleNamesCScope := pCRoleBindings.getRolesBoundToPartialBindings(cRoleKind, cRoleNameMap, csvSAs)

	// Add {Cluster}Roles bound to a ServiceAccount to either namespace- or cluster-scoped permission sets.
	for _, roleName := range inRoleNames {
		inPerms = append(inPerms, roleMap[roleName])
		delete(roleMap, roleName)
	}
	for _, roleName := range inCRoleNamesNScope {
		inPerms = append(inPerms, cRoleMap[roleName])
	}
	for _, roleName := range inCRoleNamesCScope {
		inCPerms = append(inCPerms, cRoleMap[roleName])
	}
	// Delete afterwards so both namespace- and cluster-scoped ClusterRoles can be added.
	for _, roleName := range append(inCRoleNamesNScope, inCRoleNamesCScope...) {
		delete(cRoleMap, roleName)
	}

	// Add all {Cluster}Roles not used above and all remaining bindings to out.
	for _, role := range roleMap {
		out = append(out, role)
	}
	for _, role := range cRoleMap {
		out = append(out, role)
	}
	for _, pBinding := range pRoleBindings {
		out = append(out, roleBindingMap[pBinding.Name])
	}
	for _, pBinding := range pCRoleBindings {
		out = append(out, cRoleBindingMap[pBinding.Name])
	}

	// All ServiceAccounts not in the CSV should be in out.
	for i := range c.ServiceAccounts {
		sa := c.ServiceAccounts[i]
		if _, csvHasSA := csvSAs[sa.Name]; !csvHasSA {
			out = append(out, &sa)
		}
	}

	return inPerms, inCPerms, out
}

// partialBinding is a "generic" binding.
type partialBinding struct {
	Name     string
	RoleRef  rbacv1.RoleRef
	Subjects []rbacv1.Subject
}

type partialBindings []partialBinding

// getRolesBoundToPartialBindings returns a list of role names for type refKind (one of Role, ClusterRole) in roleNameMap
// that are bound to a binding in pBindings with a ServiceAccount subject with a name in saNames.
func (pBindings *partialBindings) getRolesBoundToPartialBindings(refKind string, roleNameMap, saNames map[string]struct{}) (inNames []string) {

	for i := 0; i < len(*pBindings); i++ {
		binding := (*pBindings)[i]
		ref := binding.RoleRef
		_, hasRoleName := roleNameMap[ref.Name]
		if !hasRoleName || ref.Kind != refKind || !acceptRefGroup(ref.APIGroup) {
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
		// At least one ServiceAccount of this binding in saNames was found, so add the role's name.
		if addRole {
			inNames = append(inNames, ref.Name)
		}
		if len(binding.Subjects) == 0 && len(*pBindings) > 0 {
			*pBindings = append((*pBindings)[:i], (*pBindings)[i+1:]...)
			i--
		}
	}

	return inNames
}

func acceptRefGroup(apiGroup string) bool {
	return apiGroup == "" || apiGroup == rbacv1.SchemeGroupVersion.Group
}
