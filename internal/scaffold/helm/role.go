// Copyright 2019 The Operator-SDK Authors
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

package helm

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/scaffold"

	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/releaseutil"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

// roleDiscoveryInterface is an interface that contains just the discovery
// methods needed by the Helm role scaffold generator. Requiring just this
// interface simplifies testing.
type roleDiscoveryInterface interface {
	ServerGroupsAndResources() ([]*metav1.APIGroup, []*metav1.APIResourceList, error)
}

var DefaultRoleScaffold = scaffold.Role{
	IsClusterScoped:  false,
	SkipDefaultRules: false,
	CustomRules: []rbacv1.PolicyRule{
		// We need this rule so tiller can read namespaces to ensure they exist
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
func GenerateRoleScaffold(dc roleDiscoveryInterface, chart *chart.Chart) scaffold.Role {
	log.Info("Generating RBAC rules")

	roleScaffold := DefaultRoleScaffold
	clusterResourceRules, namespacedResourceRules, err := generateRoleRules(dc, chart)
	if err != nil {
		log.Warnf("Using default RBAC rules: failed to generate RBAC rules: %s", err)
		return roleScaffold
	}

	roleScaffold.SkipDefaultRules = true

	// Use a ClusterRole if cluster scoped resources are listed in the chart
	if len(clusterResourceRules) > 0 {
		log.Info("Scaffolding ClusterRole and ClusterRolebinding for cluster scoped resources in the helm chart")
		roleScaffold.IsClusterScoped = true
	}
	roleScaffold.CustomRules = append(roleScaffold.CustomRules, append(clusterResourceRules,
		namespacedResourceRules...)...)

	log.Warn("The RBAC rules generated in deploy/role.yaml are based on the chart's default manifest." +
		" Some rules may be missing for resources that are only enabled with custom values, and" +
		" some existing rules may be overly broad. Double check the rules generated in deploy/role.yaml" +
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
