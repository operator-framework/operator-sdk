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

// This service account exists in every namespace as the default.
const defaultServiceAccountName = "default"

func (c *Manifests) GetNonCSVObjects() (objs []controllerutil.Object) {
	// All CRDs passed in should be written.
	for i := range c.V1CustomResourceDefinitions {
		objs = append(objs, &c.V1CustomResourceDefinitions[i])
	}
	for i := range c.V1beta1CustomResourceDefinitions {
		objs = append(objs, &c.V1beta1CustomResourceDefinitions[i])
	}

	// All ServiceAccounts passed in should be written.
	for i := range c.ServiceAccounts {
		objs = append(objs, &c.ServiceAccounts[i])
	}

	// RBAC objects that are not a part of the CSV should be written separately.
	saNamesToRoleNames := make(map[string]map[string]struct{})
	for _, binding := range c.RoleBindings {
		roleRef := binding.RoleRef
		if roleRef.Kind == "Role" && (roleRef.APIGroup == "" || roleRef.APIGroup == rbacv1.SchemeGroupVersion.Group) {
			for _, name := range getSubjectServiceAccountNames(binding.Subjects) {
				if _, hasName := saNamesToRoleNames[name]; !hasName {
					saNamesToRoleNames[name] = make(map[string]struct{})
				}
				saNamesToRoleNames[name][roleRef.Name] = struct{}{}
			}
		}
	}
	// Create a list of cluster role names to ignore.
	deploymentRoleNames := make(map[string]struct{})
	for _, dep := range c.Deployments {
		saName := dep.Spec.Template.Spec.ServiceAccountName
		if saName == "" {
			saName = defaultServiceAccountName
		}
		for name := range saNamesToRoleNames[saName] {
			deploymentRoleNames[name] = struct{}{}
		}
	}
	// Add all remaining cluster roles, which are not referenced in deployments (different service account or unbound).
	for i, role := range c.Roles {
		if _, hasName := deploymentRoleNames[role.GetName()]; !hasName {
			objs = append(objs, &c.Roles[i])
		}
	}

	saNamesClusterToRoleNames := make(map[string]map[string]struct{})
	for _, binding := range c.ClusterRoleBindings {
		roleRef := binding.RoleRef
		if roleRef.Kind == "ClusterRole" && (roleRef.APIGroup == "" || roleRef.APIGroup == rbacv1.SchemeGroupVersion.Group) {
			for _, name := range getSubjectServiceAccountNames(binding.Subjects) {
				if _, hasName := saNamesClusterToRoleNames[name]; !hasName {
					saNamesClusterToRoleNames[name] = make(map[string]struct{})
				}
				saNamesClusterToRoleNames[name][roleRef.Name] = struct{}{}
			}
		}
	}
	// Create a list of cluster role names to ignore.
	deploymentClusterRoleNames := make(map[string]struct{})
	for _, dep := range c.Deployments {
		saName := dep.Spec.Template.Spec.ServiceAccountName
		if saName == "" {
			saName = "default"
		}
		for name := range saNamesClusterToRoleNames[saName] {
			deploymentClusterRoleNames[name] = struct{}{}
		}
	}
	// Add all remaining roles, which are not referenced in deployments (different service account or unbound).
	for i, clusterRole := range c.ClusterRoles {
		if _, hasName := deploymentClusterRoleNames[clusterRole.GetName()]; !hasName {
			objs = append(objs, &c.ClusterRoles[i])
		}
	}
	return objs
}

// getSubjectServiceAccountNames returns a list of all ServiceAccount subject names.
func getSubjectServiceAccountNames(subjects []rbacv1.Subject) (saNames []string) {
	for _, subject := range subjects {
		if subject.Kind == "ServiceAccount" {
			saNames = append(saNames, subject.Name)
		}
	}
	return saNames
}
