// Copyright 2019 The Operator-SDK Authors
// Modifications copyright 2020 The Operator-SDK Authors
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

package templates

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/releaseutil"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/yaml"
)

var _ file.Template = &Role{}

// Role scaffolds the config/rbac/auth_proxy_role.yaml file
type Role struct {
	file.TemplateMixin

	SkipDefaultRules bool
	CustomRules      []rbacv1.PolicyRule
}

// SetTemplateDefaults implements input.Template
func (f *Role) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("config", "rbac", "role.yaml")
	}

	f.TemplateBody = roleTemplate

	return nil
}

// todo(camilamacedo86): remove the roles added after the {{- end }}
// These roles were added because we are using the Helm pkg current implementation which
// requires the permissions for the metrics.
// More info: https://github.com/operator-framework/operator-sdk/issues/3354

const roleTemplate = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: manager-role
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
`

// roleDiscoveryInterface is an interface that contains just the discovery
// methods needed by the Helm role scaffold generator. Requiring just this
// interface simplifies testing.
type roleDiscoveryInterface interface {
	ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error)
}

var DefaultRoleScaffold = Role{
	SkipDefaultRules: false,
	CustomRules: []rbacv1.PolicyRule{
		// We need this rule so the operator can read namespaces to ensure they exist
		{
			APIGroups: []string{""},
			Resources: []string{"namespaces"},
			Verbs:     []string{"get"},
		},

		// We need this rule for leader election and release state storage to work
		{
			APIGroups: []string{""},
			Resources: []string{"configmaps", "secrets"},
			Verbs:     []string{rbacv1.VerbAll},
		},

		// We need this rule for creating Kubernetes events
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"create"},
		},
	},
}

// GenerateRoleScaffold generates a role scaffold from the provided helm chart. It
// renders a release manifest using the chart's default values and uses the Kubernetes
// discovery API to lookup each resource in the resulting manifest.
// The role scaffold will have IsClusterScoped=true if the chart lists cluster scoped resources
func GenerateRoleScaffold(dc roleDiscoveryInterface, chart *chart.Chart) Role {
	log.Info("Generating RBAC rules")

	roleScaffold := DefaultRoleScaffold
	clusterResourceRules, namespacedResourceRules, err := generateRoleRules(dc, chart)
	if err != nil {
		log.Warnf("Using default RBAC rules: failed to generate RBAC rules: %s", err)
		return roleScaffold
	}

	roleScaffold.SkipDefaultRules = true
	roleScaffold.CustomRules = append(roleScaffold.CustomRules, append(clusterResourceRules,
		namespacedResourceRules...)...)

	log.Warn("The RBAC rules generated in config/rbac/role.yaml are based on the chart's default manifest." +
		" Some rules may be missing for resources that are only enabled with custom values, and" +
		" some existing rules may be overly broad. Double check the rules generated in config/rbac/role.yaml" +
		" to ensure they meet the operator's permission requirements.")

	return roleScaffold
}

func generateRoleRules(dc roleDiscoveryInterface, chart *chart.Chart) ([]rbacv1.PolicyRule,
	[]rbacv1.PolicyRule, error) {
	_, serverResources, err := dc.ServerGroupsAndResources()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get server resources: %v", err)
	}

	manifests, err := getDefaultManifests(chart)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get default manifest: %v", err)
	}

	// Use maps of sets of resources, keyed by their group. This helps us
	// de-duplicate resources within a group as we traverse the manifests.
	clusterGroups := map[string]map[string]struct{}{}
	namespacedGroups := map[string]map[string]struct{}{}

	for _, m := range manifests {
		name := m.Name
		content := strings.TrimSpace(m.Content)

		// Ignore NOTES.txt, helper manifests, and empty manifests.
		b := filepath.Base(name)
		if b == "NOTES.txt" {
			continue
		}
		if strings.HasPrefix(b, "_") {
			continue
		}
		if content == "" || content == "---" {
			continue
		}

		// Extract the gvk from the template
		resource := unstructured.Unstructured{}
		err := yaml.Unmarshal([]byte(content), &resource)
		if err != nil {
			log.Warnf("Skipping rule generation for %s. Failed to parse manifest: %s", name, err)
			continue
		}
		groupVersion := resource.GetAPIVersion()
		group := resource.GroupVersionKind().Group
		kind := resource.GroupVersionKind().Kind

		// If we don't have the group or the kind, we won't be able to
		// create a valid role rule, log a warning and continue.
		if groupVersion == "" {
			log.Warnf("Skipping rule generation for %s. Failed to determine resource apiVersion.", name)
			continue
		}
		if kind == "" {
			log.Warnf("Skipping rule generation for %s. Failed to determine resource kind.", name)
			continue
		}

		if resourceName, namespaced, ok := getResource(serverResources, groupVersion, kind); ok {
			if !namespaced {
				if clusterGroups[group] == nil {
					clusterGroups[group] = map[string]struct{}{}
				}
				clusterGroups[group][resourceName] = struct{}{}
			} else {
				if namespacedGroups[group] == nil {
					namespacedGroups[group] = map[string]struct{}{}
				}
				namespacedGroups[group][resourceName] = struct{}{}
			}
		} else {
			log.Warnf("Skipping rule generation for %s. Failed to determine resource scope for %s.",
				name, resource.GroupVersionKind())
			continue
		}
	}

	// convert map[string]map[string]struct{} to []rbacv1.PolicyRule
	clusterRules := buildRulesFromGroups(clusterGroups)
	namespacedRules := buildRulesFromGroups(namespacedGroups)

	return clusterRules, namespacedRules, nil
}

func getDefaultManifests(c *chart.Chart) ([]releaseutil.Manifest, error) {
	install := action.NewInstall(&action.Configuration{})
	install.DryRun = true
	install.ReleaseName = "RELEASE-NAME"
	install.Replace = true
	install.ClientOnly = true
	rel, err := install.Run(c, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to render chart templates: %v", err)
	}
	_, manifests, err := releaseutil.SortManifests(releaseutil.SplitManifests(rel.Manifest),
		chartutil.DefaultVersionSet, releaseutil.InstallOrder)
	return manifests, err
}

func getResource(namespacedResourceList []*metav1.APIResourceList, groupVersion, kind string) (string, bool, bool) {
	for _, apiResourceList := range namespacedResourceList {
		if apiResourceList.GroupVersion == groupVersion {
			for _, apiResource := range apiResourceList.APIResources {
				if apiResource.Kind == kind {
					return apiResource.Name, apiResource.Namespaced, true
				}
			}
		}
	}
	return "", false, false
}

func buildRulesFromGroups(groups map[string]map[string]struct{}) []rbacv1.PolicyRule {
	rules := []rbacv1.PolicyRule{}
	for group, resourceNames := range groups {
		resources := []string{}
		for resource := range resourceNames {
			resources = append(resources, resource)
		}
		sort.Strings(resources)
		rules = append(rules, rbacv1.PolicyRule{
			APIGroups: []string{group},
			Resources: resources,
			Verbs:     []string{rbacv1.VerbAll},
		})
	}
	return rules
}

func UpdateRoleForResource(r *resource.Resource, absProjectPath string) error {
	// append rbac rule to deploy/role.yaml
	roleFilePath := filepath.Join(absProjectPath, "config", "rbac", "role.yaml")
	roleYAML, err := ioutil.ReadFile(roleFilePath)
	if err != nil {
		return fmt.Errorf("failed to read role manifest %v: %v", roleFilePath, err)
	}
	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(roleYAML, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to decode role manifest %v: %v", roleFilePath, err)
	}
	switch role := obj.(type) {
	case *rbacv1.ClusterRole:
		pr := &rbacv1.PolicyRule{}
		apiGroupFound := false
		for i := range role.Rules {
			if role.Rules[i].APIGroups[0] == r.Domain {
				apiGroupFound = true
				pr = &role.Rules[i]
				break
			}
		}
		// check if the resource already exists
		for _, resource := range pr.Resources {
			if resource == r.Plural {
				log.Infof("RBAC rules in deploy/role.yaml already up to date for the resource (%s/%s, %v)", r.Group, r.Version, r.Kind)
				return nil
			}
		}

		pr.Resources = append(pr.Resources, r.Plural)
		// create a new apiGroup if not found.
		if !apiGroupFound {
			pr.APIGroups = []string{r.Domain}
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
		return errors.New("failed to parse role.yaml as a ClusterRole")
	}
}

// MergeRoleForResource merges incoming new API resource rules with existing deploy/role.yaml
func MergeRoleForResource(r *resource.Resource, absProjectPath string, roleScaffold Role) error {
	roleFilePath := filepath.Join(absProjectPath, "config", "rbac", "role.yaml")
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
		return fmt.Errorf("customRules cannot be empty for new Role at: %s/%s", r.Group, r.Version)
	}

	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode(roleYAML, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to decode role manifest %v: %v", roleFilePath, err)
	}
	switch role := obj.(type) {
	case *rbacv1.ClusterRole:
		mergedClusterRoleRules := mergeRules(role.Rules, roleScaffold)
		role.Rules = mergedClusterRoleRules
	default:
		log.Errorf("Failed to parse role.yaml as a role %v", err)
	}

	if err := updateRoleFile(obj, roleFilePath); err != nil {
		return fmt.Errorf("failed to update for resource (%s/%s, %v): %v",
			r.Group, r.Version, r.Kind, err)
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
	if err := ioutil.WriteFile(roleFilePath, data, 0664); err != nil {
		return fmt.Errorf("failed to update %v: %v", roleFilePath, err)
	}

	return nil
}
