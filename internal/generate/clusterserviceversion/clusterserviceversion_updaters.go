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
	"k8s.io/apimachinery/pkg/version"

	"github.com/operator-framework/operator-sdk/internal/generate/collector"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

// ApplyTo applies relevant manifests in c to csv, sorts the applied updates,
// and validates the result.
func ApplyTo(c *collector.Manifests, csv *operatorsv1alpha1.ClusterServiceVersion) error {
	// Apply manifests to the CSV object.
	if err := apply(c, csv); err != nil {
		return fmt.Errorf("error updating ClusterServiceVersion: %v", err)
	}

	// Set fields required by namespaced operators. This is a no-op for cluster-
	// scoped operators.
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
		return fmt.Errorf("error applying Custom Resource: %v", err)
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

// applyRoles updates strategy's permissions with the Roles in the collector.
func applyRoles(c *collector.Manifests, strategy *operatorsv1alpha1.StrategyDetailsDeployment) {
	perms := []operatorsv1alpha1.StrategyDeploymentPermissions{}
	for _, role := range c.Roles {
		perms = append(perms, operatorsv1alpha1.StrategyDeploymentPermissions{
			ServiceAccountName: role.GetName(),
			Rules:              role.Rules,
		})
	}
	strategy.Permissions = perms
}

// applyClusterRoles updates strategy's cluserPermissions with the ClusterRoles
// in the collector.
func applyClusterRoles(c *collector.Manifests, strategy *operatorsv1alpha1.StrategyDetailsDeployment) {
	perms := []operatorsv1alpha1.StrategyDeploymentPermissions{}
	for _, role := range c.ClusterRoles {
		perms = append(perms, operatorsv1alpha1.StrategyDeploymentPermissions{
			ServiceAccountName: role.GetName(),
			Rules:              role.Rules,
		})
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

// applyWebhooks updates csv's webhookDefinitions with any
// mutating and validating webhooks in the collector.
func applyWebhooks(c *collector.Manifests, csv *operatorsv1alpha1.ClusterServiceVersion) {
	webhookDescriptions := []operatorsv1alpha1.WebhookDescription{}
	for _, webhook := range c.ValidatingWebhooks {
		webhookDescriptions = append(webhookDescriptions, validatingToWebhookDescription(webhook))
	}
	for _, webhook := range c.MutatingWebhooks {
		webhookDescriptions = append(webhookDescriptions, mutatingToWebhookDescription(webhook))
	}
	csv.Spec.WebhookDefinitions = webhookDescriptions
}

// validatingToWebhookDescription transforms webhook into a WebhookDescription.
func validatingToWebhookDescription(webhook admissionregv1.ValidatingWebhook) operatorsv1alpha1.WebhookDescription {
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
	if serviceRef := webhook.ClientConfig.Service; serviceRef != nil {
		if serviceRef.Port != nil {
			description.ContainerPort = *serviceRef.Port
		}
		description.DeploymentName = strings.TrimSuffix(serviceRef.Name, "-service")
		description.WebhookPath = serviceRef.Path
	}
	return description
}

// mutatingToWebhookDescription transforms webhook into a WebhookDescription.
func mutatingToWebhookDescription(webhook admissionregv1.MutatingWebhook) operatorsv1alpha1.WebhookDescription {
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
	if serviceRef := webhook.ClientConfig.Service; serviceRef != nil {
		if serviceRef.Port != nil {
			description.ContainerPort = *serviceRef.Port
		}
		description.DeploymentName = strings.TrimSuffix(serviceRef.Name, "-service")
		description.WebhookPath = serviceRef.Path
	}
	return description
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
	results := validation.ClusterServiceVersionValidator.Validate(csv)
	for _, r := range results {
		for _, w := range r.Warnings {
			log.Warnf("ClusterServiceVersion validation: [%s] %s", w.Type, w.Detail)
		}
		for _, e := range r.Errors {
			log.Errorf("ClusterServiceVersion validation: [%s] %s", e.Type, e.Detail)
		}
		if r.HasError() {
			return errors.New("got ClusterServiceVersion validation errors")
		}
	}
	return nil
}
