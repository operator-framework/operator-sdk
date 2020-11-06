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

package clusterserviceversion

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/validation"
	"github.com/operator-framework/operator-registry/pkg/registry"
	log "github.com/sirupsen/logrus"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/version"

	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

// ApplyTo applies relevant manifests in c to csv, sorts the applied updates,
// and validates the result.
func ApplyTo(c *collector.Manifests, csv *operatorsv1alpha1.ClusterServiceVersion) error {
	// Apply manifests to the CSV object.
	if err := apply(c, csv); err != nil {
		return err
	}

	// Set fields required by namespaced operators. This is a no-op for cluster-scoped operators.
	setNamespacedFields(csv)

	// Sort all updated fields.
	sortUpdates(csv)

	return validate(csv)
}

// apply applies relevant manifests in c to csv.
func apply(c *collector.Manifests, csv *operatorsv1alpha1.ClusterServiceVersion) error {
	strategy := getCSVInstallStrategy(csv)
	switch strategy.StrategyName {
	case operatorsv1alpha1.InstallStrategyNameDeployment:
		applyRoles(c, &strategy.StrategySpec)
		applyClusterRoles(c, &strategy.StrategySpec)
		applyDeployments(c, &strategy.StrategySpec)
	}
	csv.Spec.InstallStrategy = strategy

	applyCustomResourceDefinitions(c, csv)
	if err := applyCustomResources(c, csv); err != nil {
		return fmt.Errorf("error applying Custom Resource examples to CSV %s: %v", csv.GetName(), err)
	}
	applyWebhooks(c, csv)
	return nil
}

// Get install strategy from csv.
func getCSVInstallStrategy(csv *operatorsv1alpha1.ClusterServiceVersion) operatorsv1alpha1.NamedInstallStrategy {
	// Default to a deployment strategy if none found.
	if csv.Spec.InstallStrategy.StrategyName == "" {
		csv.Spec.InstallStrategy.StrategyName = operatorsv1alpha1.InstallStrategyNameDeployment
	}
	return csv.Spec.InstallStrategy
}

// This service account exists in every namespace as the default.
const defaultServiceAccountName = "default"

// applyRoles applies Roles to strategy's permissions field by combining Roles bound to ServiceAccounts
// into one set of permissions.
func applyRoles(c *collector.Manifests, strategy *operatorsv1alpha1.StrategyDetailsDeployment) { //nolint:dupl
	objs, _ := c.SplitCSVPermissionsObjects()
	roleSet := make(map[string]*rbacv1.Role)
	for i := range objs {
		switch t := objs[i].(type) {
		case *rbacv1.Role:
			roleSet[t.GetName()] = t
		}
	}

	saToPermissions := make(map[string]operatorsv1alpha1.StrategyDeploymentPermissions)
	for _, dep := range c.Deployments {
		saName := dep.Spec.Template.Spec.ServiceAccountName
		if saName == "" {
			saName = defaultServiceAccountName
		}
		saToPermissions[saName] = operatorsv1alpha1.StrategyDeploymentPermissions{ServiceAccountName: saName}
	}

	// Collect all role names by their corresponding service accounts via bindings. This lets us
	// look up all service accounts a role is bound to and create one set of permissions per service account.
	for _, binding := range c.RoleBindings {
		if role, hasRole := roleSet[binding.RoleRef.Name]; hasRole {
			for _, subject := range binding.Subjects {
				if perm, hasSA := saToPermissions[subject.Name]; hasSA && subject.Kind == "ServiceAccount" {
					perm.Rules = append(perm.Rules, role.Rules...)
					saToPermissions[subject.Name] = perm
				}
			}
		}
	}

	// Apply relevant roles to each service account.
	perms := []operatorsv1alpha1.StrategyDeploymentPermissions{}
	for _, perm := range saToPermissions {
		if len(perm.Rules) != 0 {
			perms = append(perms, perm)
		}
	}
	strategy.Permissions = perms
}

// applyClusterRoles applies ClusterRoles to strategy's clusterPermissions field by combining ClusterRoles
// bound to ServiceAccounts into one set of clusterPermissions.
func applyClusterRoles(c *collector.Manifests, strategy *operatorsv1alpha1.StrategyDetailsDeployment) { //nolint:dupl
	objs, _ := c.SplitCSVClusterPermissionsObjects()
	roleSet := make(map[string]*rbacv1.ClusterRole)
	for i := range objs {
		switch t := objs[i].(type) {
		case *rbacv1.ClusterRole:
			roleSet[t.GetName()] = t
		}
	}

	saToPermissions := make(map[string]operatorsv1alpha1.StrategyDeploymentPermissions)
	for _, dep := range c.Deployments {
		saName := dep.Spec.Template.Spec.ServiceAccountName
		if saName == "" {
			saName = defaultServiceAccountName
		}
		saToPermissions[saName] = operatorsv1alpha1.StrategyDeploymentPermissions{ServiceAccountName: saName}
	}

	// Collect all role names by their corresponding service accounts via bindings. This lets us
	// look up all service accounts a role is bound to and create one set of permissions per service account.
	for _, binding := range c.ClusterRoleBindings {
		if role, hasRole := roleSet[binding.RoleRef.Name]; hasRole {
			for _, subject := range binding.Subjects {
				if perm, hasSA := saToPermissions[subject.Name]; hasSA && subject.Kind == "ServiceAccount" {
					perm.Rules = append(perm.Rules, role.Rules...)
					saToPermissions[subject.Name] = perm
				}
			}
		}
	}

	// Apply relevant roles to each service account.
	perms := []operatorsv1alpha1.StrategyDeploymentPermissions{}
	for _, perm := range saToPermissions {
		if len(perm.Rules) != 0 {
			perms = append(perms, perm)
		}
	}
	strategy.ClusterPermissions = perms
}

// applyDeployments updates strategy's deployments with the Deployments
// in the collector.
func applyDeployments(c *collector.Manifests, strategy *operatorsv1alpha1.StrategyDetailsDeployment) {
	depSpecs := []operatorsv1alpha1.StrategyDeploymentSpec{}
	for _, dep := range c.Deployments {
		depSpecs = append(depSpecs, operatorsv1alpha1.StrategyDeploymentSpec{
			Name: dep.GetName(),
			Spec: dep.Spec,
		})
	}
	strategy.DeploymentSpecs = depSpecs
}

const (
	// WatchNamespaceEnv is a constant for internal use.
	WatchNamespaceEnv = "WATCH_NAMESPACE"
	// TargetNamespacesRef references the target namespaces a CSV is installed in.
	// This is required by legacy project Deployments.
	TargetNamespacesRef = "metadata.annotations['olm.targetNamespaces']"
)

// setNamespacedFields sets static fields in a CSV required by namespaced
// operators.
func setNamespacedFields(csv *operatorsv1alpha1.ClusterServiceVersion) {
	for _, dep := range csv.Spec.InstallStrategy.StrategySpec.DeploymentSpecs {
		// Set WATCH_NAMESPACE if it exists in a deployment spec..
		envVar := newFieldRefEnvVar(WatchNamespaceEnv, TargetNamespacesRef)
		setContainerEnvVarIfExists(&dep.Spec, envVar)
	}
}

// setContainerEnvVarIfExists overwrites all references to ev.Name with ev.
func setContainerEnvVarIfExists(spec *appsv1.DeploymentSpec, ev corev1.EnvVar) {
	for _, c := range spec.Template.Spec.Containers {
		for i := 0; i < len(c.Env); i++ {
			if c.Env[i].Name == ev.Name {
				c.Env[i] = ev
			}
		}
	}
}

// newFieldRefEnvVar creates a new environment variable referencing fieldPath.
func newFieldRefEnvVar(name, fieldPath string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: fieldPath,
			},
		},
	}
}

// applyCustomResourceDefinitions updates csv's customresourcedefinitions.owned
// with CustomResourceDefinitions in the collector.
// customresourcedefinitions.required are left as-is, since they are
// manually-defined values.
func applyCustomResourceDefinitions(c *collector.Manifests, csv *operatorsv1alpha1.ClusterServiceVersion) {
	ownedDescs := []operatorsv1alpha1.CRDDescription{}
	descMap := map[registry.DefinitionKey]operatorsv1alpha1.CRDDescription{}
	for _, owned := range csv.Spec.CustomResourceDefinitions.Owned {
		defKey := registry.DefinitionKey{
			Name:    owned.Name,
			Version: owned.Version,
			Kind:    owned.Kind,
		}
		descMap[defKey] = owned
	}

	var defKeys []registry.DefinitionKey
	v1crdKeys := k8sutil.DefinitionsForV1CustomResourceDefinitions(c.V1CustomResourceDefinitions...)
	defKeys = append(defKeys, v1crdKeys...)
	v1beta1crdKeys := k8sutil.DefinitionsForV1beta1CustomResourceDefinitions(c.V1beta1CustomResourceDefinitions...)
	defKeys = append(defKeys, v1beta1crdKeys...)
	// crdDescriptions don't have a 'group' field.
	for i := 0; i < len(defKeys); i++ {
		defKeys[i].Group = ""
	}

	for _, defKey := range defKeys {
		if owned, ownedExists := descMap[defKey]; ownedExists {
			ownedDescs = append(ownedDescs, owned)
		} else {
			ownedDescs = append(ownedDescs, operatorsv1alpha1.CRDDescription{
				Name:    defKey.Name,
				Version: defKey.Version,
				Kind:    defKey.Kind,
			})
		}
	}

	csv.Spec.CustomResourceDefinitions.Owned = ownedDescs
}

// applyWebhooks updates csv's webhookDefinitions with any mutating and validating webhooks in the collector.
func applyWebhooks(c *collector.Manifests, csv *operatorsv1alpha1.ClusterServiceVersion) {
	webhookDescriptions := []operatorsv1alpha1.WebhookDescription{}

	for _, webhook := range c.ValidatingWebhooks {
		var validatingServiceName string
		depName, svc := findMatchingDeploymentAndServiceForWebhook(c, webhook.ClientConfig)

		if svc != nil {
			validatingServiceName = svc.GetName()
		}

		if validatingServiceName == "" && depName == "" {
			log.Infof("No service found for validating webhook %q", webhook.Name)
		} else if depName == "" {
			log.Infof("No deployment is selected by service %q for validating webhook %q", validatingServiceName, webhook.Name)
		}
		webhookDescriptions = append(webhookDescriptions, validatingToWebhookDescription(webhook, depName, svc))
	}
	for _, webhook := range c.MutatingWebhooks {
		var mutatingServiceName string
		depName, svc := findMatchingDeploymentAndServiceForWebhook(c, webhook.ClientConfig)

		if svc != nil {
			mutatingServiceName = svc.GetName()
		}

		if mutatingServiceName == "" && depName == "" {
			log.Infof("No service found for mutating webhook %q", webhook.Name)
		} else if depName == "" {
			log.Infof("No deployment is selected by service %q for mutating webhook %q", mutatingServiceName, webhook.Name)
		}
		webhookDescriptions = append(webhookDescriptions, mutatingToWebhookDescription(webhook, depName, svc))
	}
	csv.Spec.WebhookDefinitions = webhookDescriptions
}

// The default AdmissionReviewVersions set in a CSV if not set in the source webhook.
var defaultAdmissionReviewVersions = []string{"v1beta1"}

// validatingToWebhookDescription transforms webhook into a WebhookDescription.
func validatingToWebhookDescription(webhook admissionregv1.ValidatingWebhook, depName string, ws *corev1.Service) operatorsv1alpha1.WebhookDescription {
	description := operatorsv1alpha1.WebhookDescription{
		Type:                    operatorsv1alpha1.ValidatingAdmissionWebhook,
		GenerateName:            webhook.Name,
		Rules:                   webhook.Rules,
		FailurePolicy:           webhook.FailurePolicy,
		MatchPolicy:             webhook.MatchPolicy,
		ObjectSelector:          webhook.ObjectSelector,
		SideEffects:             webhook.SideEffects,
		TimeoutSeconds:          webhook.TimeoutSeconds,
		AdmissionReviewVersions: webhook.AdmissionReviewVersions,
	}
	if len(description.AdmissionReviewVersions) == 0 {
		description.AdmissionReviewVersions = defaultAdmissionReviewVersions
	}
	if description.SideEffects == nil {
		seNone := admissionregv1.SideEffectClassNone
		description.SideEffects = &seNone
	}

	if serviceRef := webhook.ClientConfig.Service; serviceRef != nil {
		var webhookServiceRefPort int32 = 443
		if serviceRef.Port != nil {
			webhookServiceRefPort = *serviceRef.Port
		}
		if ws != nil {
			for _, port := range ws.Spec.Ports {
				if webhookServiceRefPort == port.Port {
					description.ContainerPort = port.Port
					description.TargetPort = &port.TargetPort
					break
				}
			}
		}
		description.DeploymentName = depName
		if description.DeploymentName == "" {
			description.DeploymentName = strings.TrimSuffix(serviceRef.Name, "-service")
		}
		description.WebhookPath = serviceRef.Path
	}
	return description
}

// mutatingToWebhookDescription transforms webhook into a WebhookDescription.
func mutatingToWebhookDescription(webhook admissionregv1.MutatingWebhook, depName string, ws *corev1.Service) operatorsv1alpha1.WebhookDescription {
	description := operatorsv1alpha1.WebhookDescription{
		Type:                    operatorsv1alpha1.MutatingAdmissionWebhook,
		GenerateName:            webhook.Name,
		Rules:                   webhook.Rules,
		FailurePolicy:           webhook.FailurePolicy,
		MatchPolicy:             webhook.MatchPolicy,
		ObjectSelector:          webhook.ObjectSelector,
		SideEffects:             webhook.SideEffects,
		TimeoutSeconds:          webhook.TimeoutSeconds,
		AdmissionReviewVersions: webhook.AdmissionReviewVersions,
		ReinvocationPolicy:      webhook.ReinvocationPolicy,
	}
	if len(description.AdmissionReviewVersions) == 0 {
		description.AdmissionReviewVersions = defaultAdmissionReviewVersions
	}
	if description.SideEffects == nil {
		seNone := admissionregv1.SideEffectClassNone
		description.SideEffects = &seNone
	}

	if serviceRef := webhook.ClientConfig.Service; serviceRef != nil {
		var webhookServiceRefPort int32 = 443
		if serviceRef.Port != nil {
			webhookServiceRefPort = *serviceRef.Port
		}
		if ws != nil {
			for _, port := range ws.Spec.Ports {
				if webhookServiceRefPort == port.Port {
					description.ContainerPort = port.Port
					description.TargetPort = &port.TargetPort
					break
				}
			}
		}
		description.DeploymentName = depName
		if description.DeploymentName == "" {
			description.DeploymentName = strings.TrimSuffix(serviceRef.Name, "-service")
		}
		description.WebhookPath = serviceRef.Path
	}
	return description
}

// findMatchingDeploymentAndServiceForWebhook matches a Service to a webhook's client config (if it uses a service)
// then matches that Service to a Deployment by comparing label selectors (if the Service uses label selectors).
// The names of both Service and Deployment are returned if found.
func findMatchingDeploymentAndServiceForWebhook(c *collector.Manifests, wcc admissionregv1.WebhookClientConfig) (depName string, ws *corev1.Service) {
	// Return if a service reference is not specified, since a URL will be in that case.
	if wcc.Service == nil {
		return
	}

	// Find the matching service, if any. The webhook server may be externally managed
	// if no service is created by the operator.
	for i, service := range c.Services {
		if service.GetName() == wcc.Service.Name {
			ws = &c.Services[i]
			break
		}
	}
	if ws == nil {
		return
	}

	// Only ExternalName-type services cannot have selectors.
	if ws.Spec.Type == corev1.ServiceTypeExternalName {
		return
	}

	// If a selector does not exist, there is either an Endpoint or EndpointSlice object accompanying
	// the service so it should not be added to the CSV.
	if len(ws.Spec.Selector) == 0 {
		return
	}

	// Match service against pod labels, in which the webhook server will be running.
	for _, dep := range c.Deployments {
		podTemplateLabels := dep.Spec.Template.GetLabels()
		if len(podTemplateLabels) == 0 {
			continue
		}

		depName = dep.GetName()
		// Check that all labels match.
		for key, serviceValue := range ws.Spec.Selector {
			if podTemplateValue, hasKey := podTemplateLabels[key]; !hasKey || podTemplateValue != serviceValue {
				depName = ""
				break
			}
		}
		if depName != "" {
			break
		}
	}

	return depName, ws
}

// applyCustomResources updates csv's "alm-examples" annotation with the
// Custom Resources in the collector.
func applyCustomResources(c *collector.Manifests, csv *operatorsv1alpha1.ClusterServiceVersion) error {
	examples := []json.RawMessage{}
	for _, cr := range c.CustomResources {
		crBytes, err := cr.MarshalJSON()
		if err != nil {
			return err
		}
		examples = append(examples, json.RawMessage(crBytes))
	}

	examplesJSON, err := json.MarshalIndent(examples, "", "  ")
	if err != nil {
		return err
	}
	if csv.GetAnnotations() == nil {
		csv.SetAnnotations(make(map[string]string))
	}
	csv.GetAnnotations()["alm-examples"] = string(examplesJSON)

	return nil
}

// sortUpdates sorts all fields updated in csv.
// TODO(estroz): sort other modified fields.
func sortUpdates(csv *operatorsv1alpha1.ClusterServiceVersion) {
	sort.Sort(descSorter(csv.Spec.CustomResourceDefinitions.Owned))
	sort.Sort(descSorter(csv.Spec.CustomResourceDefinitions.Required))
}

// descSorter sorts a set of crdDescriptions.
type descSorter []operatorsv1alpha1.CRDDescription

var _ sort.Interface = descSorter{}

func (descs descSorter) Len() int { return len(descs) }
func (descs descSorter) Less(i, j int) bool {
	if descs[i].Name == descs[j].Name {
		if descs[i].Kind == descs[j].Kind {
			return version.CompareKubeAwareVersionStrings(descs[i].Version, descs[j].Version) > 0
		}
		return descs[i].Kind < descs[j].Kind
	}
	return descs[i].Name < descs[j].Name
}
func (descs descSorter) Swap(i, j int) { descs[i], descs[j] = descs[j], descs[i] }

// validate will validate csv using the api validation library.
// More info: https://github.com/operator-framework/api
func validate(csv *operatorsv1alpha1.ClusterServiceVersion) error {
	if csv == nil {
		return errors.New("empty ClusterServiceVersion")
	}

	hasErrors := false
	results := validation.ClusterServiceVersionValidator.Validate(csv)
	for _, r := range results {
		for _, w := range r.Warnings {
			log.Warnf("ClusterServiceVersion validation: [%s] %s", w.Type, w.Detail)
		}
		for _, e := range r.Errors {
			log.Errorf("ClusterServiceVersion validation: [%s] %s", e.Type, e.Detail)
		}
		if r.HasError() {
			hasErrors = true
		}
	}
	if hasErrors {
		return errors.New("invalid generated ClusterServiceVersion")
	}

	return nil
}
