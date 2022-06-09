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

package rbac

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	log "github.com/sirupsen/logrus"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/releaseutil"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/discovery"
	crconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/yaml"
)

var _ machinery.Template = &ManagerRole{}

var defaultRoleFile = filepath.Join("config", "rbac", "role.yaml")

// ManagerRole scaffolds the role.yaml file
type ManagerRole struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *ManagerRole) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = defaultRoleFile
	}

	f.TemplateBody = fmt.Sprintf(roleTemplate, machinery.NewMarkerFor(f.Path, rulesMarker))

	return nil
}

var _ machinery.Inserter = &ManagerRoleUpdater{}

type ManagerRoleUpdater struct {
	machinery.ResourceMixin

	Chart            *chart.Chart
	SkipDefaultRules bool
	CustomRules      []rbacv1.PolicyRule
}

func (*ManagerRoleUpdater) GetPath() string {
	return defaultRoleFile
}

func (*ManagerRoleUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile
}

const (
	rulesMarker = "rules"
)

func (f *ManagerRoleUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(defaultRoleFile, rulesMarker),
	}
}

func (f *ManagerRoleUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap, 1)

	// If resource is not being provided we are creating the file, not updating it
	if f.Resource == nil {
		return fragments
	}

	if k8sCfg, err := crconfig.GetConfig(); err != nil {
		log.Warnf("Using default RBAC rules: failed to get Kubernetes config: %s", err)
	} else if dc, err := discovery.NewDiscoveryClientForConfig(k8sCfg); err != nil {
		log.Warnf("Using default RBAC rules: failed to create Kubernetes discovery client: %s", err)
	} else {
		f.updateForChart(dc)
	}

	buf := &bytes.Buffer{}
	tmpl := template.Must(template.New("rules").Parse(rulesFragment))
	err := tmpl.Execute(buf, f)
	if err != nil {
		panic(err)
	}

	// Generate rule fragment
	rules := []string{buf.String()}

	if len(rules) != 0 {
		fragments[machinery.NewMarkerFor(defaultRoleFile, rulesMarker)] = rules
	}
	return fragments
}

const roleTemplate = `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: manager-role
rules:
##
## Base operator rules
##
# We need to get namespaces so the operator can read namespaces to ensure they exist
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
# We need to manage Helm release secrets
- apiGroups:
  - ""
  resources:
  - secrets
  verbs:
  - "*"
# We need to create events on CRs about things happening during reconciliation
- apiGroups:
  - ""
  resources:
  - events
  verbs:
  - create

%s
`

const rulesFragment = `##
## Rules for {{.Resource.QualifiedGroup}}/{{.Resource.Version}}, Kind: {{.Resource.Kind}}
##
- apiGroups:
  - {{.Resource.QualifiedGroup}}
  resources:
  - {{.Resource.Plural}}
  - {{.Resource.Plural}}/status
  - {{.Resource.Plural}}/finalizers
  verbs:
  - create
  - delete
  - get
  - list
  - patch
  - update
  - watch
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

// updateForChart updates the role scaffold from the provided helm chart. It
// renders a release manifest using the chart's default values and uses the Kubernetes
// discovery API to lookup each resource in the resulting manifest.
// The role scaffold will have IsClusterScoped=true if the chart lists cluster scoped resources
func (f *ManagerRoleUpdater) updateForChart(dc roleDiscoveryInterface) {
	fmt.Println("Generating RBAC rules")

	clusterResourceRules, namespacedResourceRules, err := generateRoleRules(dc, f.Chart)
	if err != nil {
		log.Warnf("Using default RBAC rules: failed to generate RBAC rules: %s", err)
		return
	}

	f.SkipDefaultRules = true
	f.CustomRules = append(f.CustomRules, append(clusterResourceRules,
		namespacedResourceRules...)...)

	log.Warn("The RBAC rules generated in config/rbac/role.yaml are based on the chart's default manifest." +
		" Some rules may be missing for resources that are only enabled with custom values, and" +
		" some existing rules may be overly broad. Double check the rules generated in config/rbac/role.yaml" +
		" to ensure they meet the operator's permission requirements.")
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
	install.ReleaseName = "release-name"
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
