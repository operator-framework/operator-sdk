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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/api/pkg/validation"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/version"

	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/operator-framework/operator-sdk/internal/generate/collector"
)

func applyTo(c *collector.Manifests, csv *operatorsv1alpha1.ClusterServiceVersion) error {
	// Apply manifests to the CSV object.
	if err := apply(c, csv); err != nil {
		return fmt.Errorf("error updating ClusterServiceVersion: %v", err)
	}

	// Finally sort all updated fields.
	sortUpdates(csv)

	return validateClusterServiceVersion(csv)
}

// apply applies the manifests in the collection to csv.
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

// applyRoles updates strategy's permissions with the Roles in the collection.
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
// in the collection.
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
// in the collection.
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

// applyCustomResourceDefinitions updates csv's customresourcedefinitions.owned
// with CustomResourceDefinitions in the collection.
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
	for _, crd := range c.CustomResourceDefinitions {
		for _, ver := range crd.Spec.Versions {
			defKey := registry.DefinitionKey{
				Name:    crd.GetName(),
				Version: ver.Name,
				Kind:    crd.Spec.Names.Kind,
			}
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
	}
	csv.Spec.CustomResourceDefinitions.Owned = ownedDescs
}

// applyCustomResources updates csv's "alm-examples" annotation with the
// Custom Resources in the collection.
func applyCustomResources(c *collector.Manifests, csv *operatorsv1alpha1.ClusterServiceVersion) error {
	examples := []json.RawMessage{}
	for _, cr := range c.CustomResources {
		crBytes, err := cr.MarshalJSON()
		if err != nil {
			return err
		}
		examples = append(examples, json.RawMessage(crBytes))
	}
	examplesJSON, err := json.Marshal(examples)
	if err != nil {
		return err
	}
	examplesJSON, err = prettifyJSON(examplesJSON)
	if err != nil {
		return err
	}
	if csv.GetAnnotations() == nil {
		csv.SetAnnotations(make(map[string]string))
	}
	csv.GetAnnotations()["alm-examples"] = string(examplesJSON)
	return nil
}

// prettifyJSON returns a JSON in a pretty format
func prettifyJSON(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")
	return out.Bytes(), err
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

// validateClusterServiceVersion will validate csv using the api validation library.
// More info: https://github.com/operator-framework/api
func validateClusterServiceVersion(csv *operatorsv1alpha1.ClusterServiceVersion) error {
	if csv == nil {
		return errors.New("empty ClusterServiceVersion")
	}
	results := validation.ClusterServiceVersionValidator.Validate(csv)
	for _, r := range results {
		if r.HasError() {
			for _, e := range r.Errors {
				log.Errorf("ClusterServiceVersion validation: [%s] %s", e.Type, e.Detail)
			}
			return errors.New("got ClusterServiceVersion validation errors")
		}
		for _, w := range r.Warnings {
			log.Warnf("ClusterServiceVersion validation: [%s] %s", w.Type, w.Detail)
		}
	}
	return nil
}
