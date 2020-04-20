// Copyright 2018 The Operator-SDK Authors
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

package scaffold

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"

	log "github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	cgoscheme "k8s.io/client-go/kubernetes/scheme"
	yaml "sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
)

const RoleYamlFile = "role.yaml"

type Role struct {
	input.Input

	IsClusterScoped  bool
	SkipDefaultRules bool
	CustomRules      []rbacv1.PolicyRule
}

func (s *Role) GetInput() (input.Input, error) {
	if s.Path == "" {
		s.Path = filepath.Join(DeployDir, RoleYamlFile)
	}
	s.TemplateBody = roleTemplate
	return s.Input, nil
}

//nolint:lll //references are not to be sliced
func UpdateRoleForResource(r *Resource, absProjectPath string) error {
	// append rbac rule to deploy/role.yaml
	roleFilePath := filepath.Join(absProjectPath, DeployDir, RoleYamlFile)
	roleYAML, err := ioutil.ReadFile(roleFilePath)
	if err != nil {
		return fmt.Errorf("failed to read role manifest %v: %v", roleFilePath, err)
	}
	obj, _, err := cgoscheme.Codecs.UniversalDeserializer().Decode(roleYAML, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to decode role manifest %v: %v", roleFilePath, err)
	}
	switch role := obj.(type) {
	case *rbacv1.Role:
		pr := &rbacv1.PolicyRule{}

		apiGroupFound := false
		for i := range role.Rules {
			if role.Rules[i].APIGroups[0] == r.FullGroup {
				apiGroupFound = true
				pr = &role.Rules[i]
				break
			}
		}

		// check if the resource already exists
		for _, resource := range pr.Resources {
			if resource == r.Resource {
				log.Infof("RBAC rules in deploy/role.yaml already up to date for the resource (%v, %v)", r.APIVersion, r.Kind)
				return nil
			}
		}

		pr.Resources = append(pr.Resources, r.Resource)
		// create a new apiGroup if not found.
		if !apiGroupFound {
			pr.APIGroups = []string{r.FullGroup}
			// Using "*" to allow access to the resource and all its subresources e.g "memcacheds" and "memcacheds/finalizers"
			// https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#ownerreferencespermissionenforcement
			pr.Resources = []string{"*"}
			pr.Verbs = []string{
				"create",
				"delete",
				"get",
				"list",
				"patch",
				"update",
				"watch",
			}
			role.Rules = append(role.Rules, *pr)
		}

		return updateRoleFile(&role, roleFilePath)
	case *rbacv1.ClusterRole:
		pr := &rbacv1.PolicyRule{}
		apiGroupFound := false
		for i := range role.Rules {
			if role.Rules[i].APIGroups[0] == r.FullGroup {
				apiGroupFound = true
				pr = &role.Rules[i]
				break
			}
		}
		// check if the resource already exists
		for _, resource := range pr.Resources {
			if resource == r.Resource {
				log.Infof("RBAC rules in deploy/role.yaml already up to date for the resource (%v, %v)", r.APIVersion, r.Kind)
				return nil
			}
		}

		pr.Resources = append(pr.Resources, r.Resource)
		// create a new apiGroup if not found.
		if !apiGroupFound {
			pr.APIGroups = []string{r.FullGroup}
			// Using "*" to allow access to the resource and all its subresources e.g "memcacheds" and "memcacheds/finalizers"
			// https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#ownerreferencespermissionenforcement
			pr.Resources = []string{"*"}
			pr.Verbs = []string{
				"create",
				"delete",
				"get",
				"list",
				"patch",
				"update",
				"watch",
			}
			role.Rules = append(role.Rules, *pr)
		}

		return updateRoleFile(&role, roleFilePath)
	default:
		return errors.New("failed to parse role.yaml as a role")
	}
}

// MergeRoleForResource merges incoming new API resource rules with existing deploy/role.yaml
func MergeRoleForResource(r *Resource, absProjectPath string, roleScaffold Role) error {
	roleFilePath := filepath.Join(absProjectPath, DeployDir, RoleYamlFile)
	roleYAML, err := ioutil.ReadFile(roleFilePath)
	if err != nil {
		return fmt.Errorf("failed to read role manifest %v: %v", roleFilePath, err)
	}
	// Check for existing role.yaml
	if len(roleYAML) == 0 {
		return fmt.Errorf("empty Role File at: %v", absProjectPath)
	}
	// Check for incoming new Role
	if len(roleScaffold.CustomRules) == 0 {
		return fmt.Errorf("customRules cannot be empty for new Role at: %v", r.APIVersion)
	}

	obj, _, err := cgoscheme.Codecs.UniversalDeserializer().Decode(roleYAML, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to decode role manifest %v: %v", roleFilePath, err)
	}
	switch role := obj.(type) {
	case *rbacv1.Role:
		// TODO: Add logic to merge Cluster scoped rules into existing Kind: Role scoped rules
		// Error out for ClusterRole merging with existing Kind: Role
		if roleScaffold.IsClusterScoped {
			return fmt.Errorf("cannot Merge Cluster scoped rules with existing deploy/role.yaml. " +
				"please modify existing deploy/role.yaml and deploy/role_binding.yaml " +
				"to reflect Cluster scope and try again")
		}
		mergedRoleRules := mergeRules(role.Rules, roleScaffold)
		role.Rules = mergedRoleRules
	case *rbacv1.ClusterRole:
		mergedClusterRoleRules := mergeRules(role.Rules, roleScaffold)
		role.Rules = mergedClusterRoleRules
	default:
		log.Errorf("Failed to parse role.yaml as a role %v", err)
	}

	if err := updateRoleFile(obj, roleFilePath); err != nil {
		return fmt.Errorf("failed to update for resource (%v, %v): %v",
			r.APIVersion, r.Kind, err)
	}

	return UpdateRoleForResource(r, absProjectPath)
}

func ifMatches(pr []string, newPr []string) bool {

	sort.Strings(pr)
	sort.Strings(newPr)

	if len(pr) != len(newPr) {
		return false
	}
	for i, v := range pr {
		if v != newPr[i] {
			return false
		}
	}
	return true
}

func findResource(resources []string, search string) bool {
	for _, r := range resources {
		if r == search || r == "*" {
			return true
		}
	}
	return false
}

func mergeRules(rules1 []rbacv1.PolicyRule, rules2 Role) []rbacv1.PolicyRule {
	for j := range rules2.CustomRules {
		ruleFound := false
		prj := &rules2.CustomRules[j]
	iLoop:
		for i, pri := range rules1 {
			// check if apiGroup, verbs, resourceName, and nonResourceURLS matches for new resource.
			apiGroupsEqual := ifMatches(pri.APIGroups, prj.APIGroups)
			verbsEqual := ifMatches(pri.Verbs, prj.Verbs)
			resourceNamesEqual := ifMatches(pri.ResourceNames, prj.ResourceNames)
			nonResourceURLsEqual := ifMatches(pri.NonResourceURLs, prj.NonResourceURLs)

			if apiGroupsEqual && verbsEqual && resourceNamesEqual && nonResourceURLsEqual {
				for _, newResource := range prj.Resources {
					if !findResource(pri.Resources, newResource) {
						// append rbac rule to deploy/role.yaml
						rules1[i].Resources = append(rules1[i].Resources, newResource)
					}
				}
				ruleFound = true
				break iLoop
			}
		}
		if !ruleFound {
			rules1 = append(rules1, *prj)
		}
	}
	return rules1
}
func updateRoleFile(role interface{}, roleFilePath string) error {
	data, err := yaml.Marshal(&role)
	if err != nil {
		return fmt.Errorf("failed to marshal role(%+v): %v", role, err)
	}
	if err := ioutil.WriteFile(roleFilePath, data, fileutil.DefaultFileMode); err != nil {
		return fmt.Errorf("failed to update %v: %v", roleFilePath, err)
	}

	return nil
}

const roleTemplate = `kind: {{if .IsClusterScoped}}Cluster{{end}}Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: {{.ProjectName}}
rules:
{{- if not .SkipDefaultRules }}
- apiGroups:
  - ""
  resources:
  - pods
  - services
  - services/finalizers
  - endpoints
  - persistentvolumeclaims
  - events
  - configmaps
  - secrets
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
- apiGroups:
  - apps
  resources:
  - deployments
  - daemonsets
  - replicasets
  - statefulsets
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
{{- end }}
{{- range .CustomRules }}
- verbs:
  {{- range .Verbs }}
  - "{{ . }}"
  {{- end }}
  {{- with .APIGroups }}
  apiGroups:
  {{- range . }}
  - "{{ . }}"
  {{- end }}
  {{- end }}
  {{- with .Resources }}
  resources:
  {{- range . }}
  - "{{ . }}"
  {{- end }}
  {{- end }}
  {{- with .ResourceNames }}
  resourceNames:
  {{- range . }}
  - "{{ . }}"
  {{- end }}
  {{- end }}
  {{- with .NonResourceURLs }}
  nonResourceURLs:
  {{- range . }}
  - "{{ . }}"
  {{- end }}
  {{- end }}
{{- end }}
- apiGroups:
  - monitoring.coreos.com
  resources:
  - servicemonitors
  verbs:
  - "get"
  - "create"
- apiGroups:
  - apps
  resources:
  - deployments/finalizers
  resourceNames:
  - {{ .ProjectName }}
  verbs:
  - "update"
- apiGroups:
  - ""
  resources:
  - pods
  verbs:
  - get
- apiGroups:
  - apps
  resources:
  - replicasets
  - deployments
  verbs:
  - get
`
