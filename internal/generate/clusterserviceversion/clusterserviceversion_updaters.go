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
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

// ApplyTo applies relevant manifests in c to csv, sorts the applied updates,
// and validates the result.
func ApplyTo(c *collector.Manifests, csv *operatorsv1alpha1.ClusterServiceVersion, extraSAs []string) error {
	// Apply manifests to the CSV object.
	if err := apply(c, csv, extraSAs); err != nil {
		return err
	}

	// Set fields required by namespaced operators. This is a no-op for cluster-scoped operators.
	setNamespacedFields(csv)

	return validate(csv)
}

// apply applies relevant manifests in c to csv.
func apply(c *collector.Manifests, csv *operatorsv1alpha1.ClusterServiceVersion, extraSAs []string) error {
	strategy := getCSVInstallStrategy(csv)
	switch strategy.StrategyName {
	case operatorsv1alpha1.InstallStrategyNameDeployment:
		inPerms, inCPerms, _ := c.SplitCSVPermissionsObjects(extraSAs)
		applyRoles(c, inPerms, &strategy.StrategySpec, extraSAs)
		applyClusterRoles(c, inCPerms, &strategy.StrategySpec, extraSAs)
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
func applyRoles(c *collector.Manifests, objs []client.Object, strategy *operatorsv1alpha1.StrategyDetailsDeployment, extraSAs []string) { //nolint:dupl
	roleSet := make(map[string]rbacv1.Role)
	cRoleSet := make(map[string]rbacv1.ClusterRole)
	for i := range objs {
		switch t := objs[i].(type) {
		case *rbacv1.Role:
			roleSet[t.GetName()] = *t
		case *rbacv1.ClusterRole:
			cRoleSet[t.GetName()] = *t
		}
	}

	// Collect all role and cluster role names by their corresponding service accounts via bindings. This lets us
	// look up all service accounts a role is bound to and create one set of permissions per service account.
	saToPermissions := initPermissionSet(c.Deployments, extraSAs)
	for _, binding := range c.RoleBindings {
		for _, subject := range binding.Subjects {
			perm, hasSA := saToPermissions[subject.Name]
			if subject.Kind != "ServiceAccount" || !hasSA {
				continue
			}
			var (
				rules   []rbacv1.PolicyRule
				hasRole bool
			)
			switch binding.RoleRef.Kind {
			case "Role":
				role, has := roleSet[binding.RoleRef.Name]
				rules = role.Rules
				hasRole = has
			case "ClusterRole":
				role, has := cRoleSet[binding.RoleRef.Name]
				rules = role.Rules
				hasRole = has
			default:
				continue
			}
			if hasRole {
				perm.Rules = append(perm.Rules, rules...)
				saToPermissions[subject.Name] = perm
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
	sort.Slice(perms, func(i, j int) bool {
		return perms[i].ServiceAccountName < perms[j].ServiceAccountName
	})
	strategy.Permissions = perms
}

// applyClusterRoles applies ClusterRoles to strategy's clusterPermissions field by combining ClusterRoles
// bound to ServiceAccounts into one set of clusterPermissions.
func applyClusterRoles(c *collector.Manifests, objs []client.Object, strategy *operatorsv1alpha1.StrategyDetailsDeployment, extraSAs []string) { //nolint:dupl
	roleSet := make(map[string]rbacv1.ClusterRole)
	for i := range objs {
		switch t := objs[i].(type) {
		case *rbacv1.ClusterRole:
			roleSet[t.GetName()] = *t
		}
	}

	// Collect all role names by their corresponding service accounts via bindings. This lets us
	// look up all service accounts a role is bound to and create one set of permissions per service account.
	saToPermissions := initPermissionSet(c.Deployments, extraSAs)
	for _, binding := range c.ClusterRoleBindings {
		for _, subject := range binding.Subjects {
			perm, hasSA := saToPermissions[subject.Name]
			if !hasSA || subject.Kind != "ServiceAccount" {
				continue
			}
			if role, hasRole := roleSet[binding.RoleRef.Name]; hasRole {
				perm.Rules = append(perm.Rules, role.Rules...)
				saToPermissions[subject.Name] = perm
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
	sort.Slice(perms, func(i, j int) bool {
		return perms[i].ServiceAccountName < perms[j].ServiceAccountName
	})
	strategy.ClusterPermissions = perms
}

// initPermissionSet initializes a map of ServiceAccount name to permissions, which are empty.
func initPermissionSet(deps []appsv1.Deployment, extraSAs []string) map[string]operatorsv1alpha1.StrategyDeploymentPermissions {
	saToPermissions := make(map[string]operatorsv1alpha1.StrategyDeploymentPermissions)
	for _, dep := range deps {
		saName := dep.Spec.Template.Spec.ServiceAccountName
		if saName == "" {
			saName = defaultServiceAccountName
		}
		saToPermissions[saName] = operatorsv1alpha1.StrategyDeploymentPermissions{ServiceAccountName: saName}
	}
	for _, extraSA := range extraSAs {
		saToPermissions[extraSA] = operatorsv1alpha1.StrategyDeploymentPermissions{ServiceAccountName: extraSA}
	}
	return saToPermissions
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

	for _, svc := range c.Services {
		crdToConfigMap := getConvWebhookCRDNamesAndConfig(c, svc.GetName())

		if len(crdToConfigMap) != 0 {
			depName := findMatchingDepNameFromService(c, &svc)
			des := conversionToWebhookDescription(crdToConfigMap, depName, &svc)
			webhookDescriptions = append(webhookDescriptions, des...)
		}
	}
	// Sorts the WebhookDescriptions based on natural order of webhookDescriptions Type
	sort.Slice(webhookDescriptions, func(i, j int) bool {
		return webhookDescriptions[i].GenerateName < webhookDescriptions[j].GenerateName
	})
	csv.Spec.WebhookDefinitions = webhookDescriptions
}

// conversionToWebhookDescription takes in a map of {crdNames, apiextv.WebhookConversion} and groups
// all the crds with same port and path. It then creates a webhook description for each unique combination of
// port and path.
// For example: if we have the following map: {crd1:[portX+pathX], crd2: [portX+pathX], crd3: [portY:partY]},
// we will create 2 webhook descriptions: one with [portX+pathX]:[crd1, crd2] and the other with [portY:pathY]:[crd3]
func conversionToWebhookDescription(crdToConfig map[string]apiextv1.WebhookConversion, depName string, ws *corev1.Service) []operatorsv1alpha1.WebhookDescription {
	des := make([]operatorsv1alpha1.WebhookDescription, 0)

	// this is a map of serviceportAndPath configs, and the respective CRDs.
	webhookDescriptions := crdGroups(crdToConfig)

	for serviceConfig, crds := range webhookDescriptions {
		// we need this to get the conversionReviewVersions.
		// here, we assume all crds having same servicePortAndPath config will have
		// same conversion review versions.
		config, ok := crdToConfig[crds[0]]
		if !ok {
			log.Infof("Webhook config for CRD %q not found", crds[0])
			continue
		}

		description := operatorsv1alpha1.WebhookDescription{
			Type:                    operatorsv1alpha1.ConversionWebhook,
			ConversionCRDs:          crds,
			AdmissionReviewVersions: config.ConversionReviewVersions,
			WebhookPath:             &serviceConfig.Path,
			DeploymentName:          depName,
			GenerateName:            getGenerateName(crds),
			SideEffects: func() *admissionregv1.SideEffectClass {
				seNone := admissionregv1.SideEffectClassNone
				return &seNone
			}(),
		}

		if len(description.AdmissionReviewVersions) == 0 {
			log.Infof("ConversionReviewVersion not found for the deployment %q", depName)
		}

		var webhookServiceRefPort int32 = 443

		if serviceConfig.Port != nil {
			webhookServiceRefPort = *serviceConfig.Port
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

		if description.DeploymentName == "" {
			if config.ClientConfig.Service != nil {
				description.DeploymentName = strings.TrimSuffix(config.ClientConfig.Service.Name, "-service")
			}
		}

		description.WebhookPath = &serviceConfig.Path
		des = append(des, description)
	}

	return des
}

// serviceportPath is refers to the group of webhook service and
// path names and port.
type serviceportPath struct {
	Port *int32
	Path string
}

// crdGroups groups the crds with similar service port and name. It returns a map of serviceportPath
// and the corresponding crd names.
func crdGroups(crdToConfig map[string]apiextv1.WebhookConversion) map[serviceportPath][]string {

	uniqueConfig := make(map[serviceportPath][]string)

	for crdName, config := range crdToConfig {
		serviceportPath := serviceportPath{
			Port: config.ClientConfig.Service.Port,
			Path: *config.ClientConfig.Service.Path,
		}

		uniqueConfig[serviceportPath] = append(uniqueConfig[serviceportPath], crdName)
	}

	return uniqueConfig
}

func getConvWebhookCRDNamesAndConfig(c *collector.Manifests, serviceName string) map[string]apiextv1.WebhookConversion {
	if serviceName == "" {
		return nil
	}

	crdToConfig := make(map[string]apiextv1.WebhookConversion)

	for _, crd := range c.V1CustomResourceDefinitions {
		if crd.Spec.Conversion != nil {
			whConv := crd.Spec.Conversion.Webhook
			if whConv != nil && whConv.ClientConfig != nil && whConv.ClientConfig.Service != nil {
				if whConv.ClientConfig.Service.Name == serviceName {
					crdToConfig[crd.GetName()] = *whConv
				}
			}
		}
	}

	for _, crd := range c.V1beta1CustomResourceDefinitions {
		whConv := crd.Spec.Conversion
		if whConv != nil && whConv.WebhookClientConfig != nil && whConv.WebhookClientConfig.Service != nil {
			if whConv.WebhookClientConfig.Service.Name == serviceName {
				v1whConv := apiextv1.WebhookConversion{
					ClientConfig:             &apiextv1.WebhookClientConfig{Service: &apiextv1.ServiceReference{}},
					ConversionReviewVersions: crd.Spec.Conversion.ConversionReviewVersions,
				}
				if path := whConv.WebhookClientConfig.Service.Path; path != nil {
					v1whConv.ClientConfig.Service.Path = new(string)
					*v1whConv.ClientConfig.Service.Path = *path
				}
				crdToConfig[crd.GetName()] = v1whConv
			}
		}
	}
	return crdToConfig
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
// and uses that service to find the deployment name.
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

	depName = findMatchingDepNameFromService(c, ws)

	return depName, ws
}

// findMatchingDepNameFromService matches the provided service to a deployment by comparing label selectors (if
// Service uses label selectors).
func findMatchingDepNameFromService(c *collector.Manifests, ws *corev1.Service) (depName string) {
	// Match service against pod labels, in which the webhook server will be running
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
	return depName
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

// generateName takes in a list of crds, and returns a conversion webhook generator name.
func getGenerateName(crds []string) string {
	sort.Strings(crds)
	joinedResourceNames := strings.Builder{}

	for _, name := range crds {
		if name != "" {
			joinedResourceNames.WriteString(strings.Split(name, ".")[0])
		}
	}
	return fmt.Sprintf("c%s.kb.io", joinedResourceNames.String())
}
