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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// TODO(estroz): there's a significant amount of code dupliation here, a byproduct of Go's type system.
// However at least a few bits can be refactored so each method is smaller.

const (
	// This service account exists in every namespace as the default.
	defaultServiceAccountName = "default"

	serviceAccountKind = "ServiceAccount"
)

// SplitCSVPermissionsObjects splits roles that should be written to a CSV as permissions (in)
// from roles and role bindings that should be written directly to the bundle (out).
func (c *Manifests) SplitCSVPermissionsObjects() (in, out []controllerutil.Object) { //nolint:dupl
	roleMap := make(map[string]*rbacv1.Role)
	for i := range c.Roles {
		roleMap[c.Roles[i].GetName()] = &c.Roles[i]
	}
	roleBindingMap := make(map[string]*rbacv1.RoleBinding)
	for i := range c.RoleBindings {
		roleBindingMap[c.RoleBindings[i].GetName()] = &c.RoleBindings[i]
	}

	// Check for unbound roles.
	for roleName, role := range roleMap {
		hasRef := false
		for _, roleBinding := range roleBindingMap {
			roleRef := roleBinding.RoleRef
			if roleRef.Kind == "Role" && (roleRef.APIGroup == "" || roleRef.APIGroup == rbacv1.SchemeGroupVersion.Group) {
				if roleRef.Name == roleName {
					hasRef = true
					break
				}
			}
		}
		if !hasRef {
			out = append(out, role)
			delete(roleMap, roleName)
		}
	}

	// If a role is bound and:
	// 1. the binding only has one subject and it is a service account that maps to a deployment service account,
	//    add the role to in.
	// 2. the binding only has one subject and it does not map to a deployment service account or is not a service account,
	//    add both role and binding to out.
	// 3. the binding has more than one subject and:
	//    a. one of those subjects is a deployment's service account, add both role and binding to out and role to in.
	//    b. none of those subjects is a service account or maps to a deployment's service account, add both role and binding to out.
	deploymentSANames := make(map[string]struct{})
	for _, dep := range c.Deployments {
		saName := dep.Spec.Template.Spec.ServiceAccountName
		if saName == "" {
			saName = defaultServiceAccountName
		}
		deploymentSANames[saName] = struct{}{}
	}

	inRoleNames := make(map[string]struct{})
	outRoleNames := make(map[string]struct{})
	outRoleBindingNames := make(map[string]struct{})
	for _, binding := range c.RoleBindings {
		roleRef := binding.RoleRef
		if roleRef.Kind == "Role" && (roleRef.APIGroup == "" || roleRef.APIGroup == rbacv1.SchemeGroupVersion.Group) {
			numSubjects := len(binding.Subjects)
			if numSubjects == 1 {
				// cases (1) and (2).
				if _, hasSA := deploymentSANames[binding.Subjects[0].Name]; hasSA && binding.Subjects[0].Kind == serviceAccountKind {
					inRoleNames[roleRef.Name] = struct{}{}
				} else {
					outRoleNames[roleRef.Name] = struct{}{}
					outRoleBindingNames[binding.GetName()] = struct{}{}
				}
			} else {
				// case (3).
				for _, subject := range binding.Subjects {
					if _, hasSA := deploymentSANames[subject.Name]; hasSA && subject.Kind == serviceAccountKind {
						// case (3a).
						inRoleNames[roleRef.Name] = struct{}{}
					}
				}
				// case (3b).
				outRoleNames[roleRef.Name] = struct{}{}
				outRoleBindingNames[binding.GetName()] = struct{}{}
			}
		}
	}

	for roleName := range inRoleNames {
		if role, hasRoleName := roleMap[roleName]; hasRoleName {
			in = append(in, role)
		}
	}
	for roleName := range outRoleNames {
		if role, hasRoleName := roleMap[roleName]; hasRoleName {
			out = append(out, role)
		}
	}
	for roleBindingName := range outRoleBindingNames {
		if roleBinding, hasRoleBindingName := roleBindingMap[roleBindingName]; hasRoleBindingName {
			out = append(out, roleBinding)
		}
	}

	return in, out
}

// SplitCSVClusterPermissionsObjects splits cluster roles that should be written to a CSV as clusterPermissions (in)
// from cluster roles and cluster role bindings that should be written directly to the bundle (out).
func (c *Manifests) SplitCSVClusterPermissionsObjects() (in, out []controllerutil.Object) { //nolint:dupl
	roleMap := make(map[string]*rbacv1.ClusterRole)
	for i := range c.ClusterRoles {
		roleMap[c.ClusterRoles[i].GetName()] = &c.ClusterRoles[i]
	}
	roleBindingMap := make(map[string]*rbacv1.ClusterRoleBinding)
	for i := range c.ClusterRoleBindings {
		roleBindingMap[c.ClusterRoleBindings[i].GetName()] = &c.ClusterRoleBindings[i]
	}

	// Check for unbound roles.
	for roleName, role := range roleMap {
		hasRef := false
		for _, roleBinding := range roleBindingMap {
			roleRef := roleBinding.RoleRef
			if roleRef.Kind == "ClusterRole" && (roleRef.APIGroup == "" || roleRef.APIGroup == rbacv1.SchemeGroupVersion.Group) {
				if roleRef.Name == roleName {
					hasRef = true
					break
				}
			}
		}
		if !hasRef {
			out = append(out, role)
			delete(roleMap, roleName)
		}
	}

	// If a role is bound and:
	// 1. the binding only has one subject and it is a service account that maps to a deployment service account,
	//    add the role to in.
	// 2. the binding only has one subject and it does not map to a deployment service account or is not a service account,
	//    add both role and binding to out.
	// 3. the binding has more than one subject and:
	//    a. one of those subjects is a deployment's service account, add both role and binding to out and role to in.
	//    b. none of those subjects is a service account or maps to a deployment's service account, add both role and binding to out.
	deploymentSANames := make(map[string]struct{})
	for _, dep := range c.Deployments {
		saName := dep.Spec.Template.Spec.ServiceAccountName
		if saName == "" {
			saName = defaultServiceAccountName
		}
		deploymentSANames[saName] = struct{}{}
	}

	inRoleNames := make(map[string]struct{})
	outRoleNames := make(map[string]struct{})
	outRoleBindingNames := make(map[string]struct{})
	for _, binding := range c.ClusterRoleBindings {
		roleRef := binding.RoleRef
		if roleRef.Kind == "ClusterRole" && (roleRef.APIGroup == "" || roleRef.APIGroup == rbacv1.SchemeGroupVersion.Group) {
			numSubjects := len(binding.Subjects)
			if numSubjects == 1 {
				// cases (1) and (2).
				if _, hasSA := deploymentSANames[binding.Subjects[0].Name]; hasSA && binding.Subjects[0].Kind == serviceAccountKind {
					inRoleNames[roleRef.Name] = struct{}{}
				} else {
					outRoleNames[roleRef.Name] = struct{}{}
					outRoleBindingNames[binding.GetName()] = struct{}{}
				}
			} else {
				// case (3).
				for _, subject := range binding.Subjects {
					if _, hasSA := deploymentSANames[subject.Name]; hasSA && subject.Kind == serviceAccountKind {
						// case (3a).
						inRoleNames[roleRef.Name] = struct{}{}
					}
				}
				// case (3b).
				outRoleNames[roleRef.Name] = struct{}{}
				outRoleBindingNames[binding.GetName()] = struct{}{}
			}
		}
	}

	for roleName := range inRoleNames {
		if role, hasRoleName := roleMap[roleName]; hasRoleName {
			in = append(in, role)
		}
	}
	for roleName := range outRoleNames {
		if role, hasRoleName := roleMap[roleName]; hasRoleName {
			out = append(out, role)
		}
	}
	for roleBindingName := range outRoleBindingNames {
		if roleBinding, hasRoleBindingName := roleBindingMap[roleBindingName]; hasRoleBindingName {
			out = append(out, roleBinding)
		}
	}

	return in, out
}
