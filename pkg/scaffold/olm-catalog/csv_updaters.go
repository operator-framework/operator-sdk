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

package catalog

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	olmApi "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olmInstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

// CSVUpdater is an interface for any data that can be in a CSV, which will be
// set to the corresponding field on Apply().
type CSVUpdater interface {
	// Apply applies a data update to a CSV argument.
	Apply(*olmApi.ClusterServiceVersion) error
}

type updaterStore struct {
	installStrategy *CSVInstallStrategyUpdate
	crdUpdate       *CSVCustomResourceDefinitionsUpdate
}

func NewUpdaterStore() *updaterStore {
	return &updaterStore{
		installStrategy: &CSVInstallStrategyUpdate{
			&olmInstall.StrategyDetailsDeployment{},
		},
		crdUpdate: &CSVCustomResourceDefinitionsUpdate{
			&olmApi.CustomResourceDefinitions{},
		},
	}
}

// Apply iteratively calls each stored CSVUpdater's Apply() method.
func (s *updaterStore) Apply(csv *olmApi.ClusterServiceVersion) error {
	for _, updater := range []CSVUpdater{s.installStrategy, s.crdUpdate} {
		if err := updater.Apply(csv); err != nil {
			return err
		}
	}
	return nil
}

func getKindfromYAML(yamlData []byte) (string, error) {
	// Get Kind for inital categorization.
	var temp struct {
		Kind string
	}
	if err := yaml.Unmarshal(yamlData, &temp); err != nil {
		return "", err
	}
	return temp.Kind, nil
}

func (s *updaterStore) AddToUpdater(yamlSpec []byte) error {
	kind, err := getKindfromYAML(yamlSpec)
	if err != nil {
		return err
	}

	switch kind {
	case "Role":
		return s.AddRole(yamlSpec)
	case "ClusterRole":
		return s.AddClusterRole(yamlSpec)
	case "Deployment":
		return s.AddDeploymentSpec(yamlSpec)
	case "CustomResourceDefinition":
		// TODO: determine whether 'owned' or 'required'
		return s.AddOwnedCRD(yamlSpec)
	}
	return nil
}

type CSVInstallStrategyUpdate struct {
	*olmInstall.StrategyDetailsDeployment
}

func (store *updaterStore) AddRole(yamlDoc []byte) error {
	role := &rbacv1.Role{}
	if err := yaml.Unmarshal(yamlDoc, role); err != nil {
		return err
	}
	perm := olmInstall.StrategyDeploymentPermissions{
		ServiceAccountName: role.ObjectMeta.Name,
		Rules:              role.Rules,
	}
	store.installStrategy.Permissions = append(store.installStrategy.Permissions, perm)

	return nil
}

func (store *updaterStore) AddClusterRole(yamlDoc []byte) error {
	clusterRole := &rbacv1.ClusterRole{}
	if err := yaml.Unmarshal(yamlDoc, clusterRole); err != nil {
		return err
	}
	perm := olmInstall.StrategyDeploymentPermissions{
		ServiceAccountName: clusterRole.ObjectMeta.Name,
		Rules:              clusterRole.Rules,
	}
	store.installStrategy.ClusterPermissions = append(store.installStrategy.ClusterPermissions, perm)

	return nil
}

func (store *updaterStore) AddDeploymentSpec(yamlDoc []byte) error {
	dep := &appsv1.Deployment{}
	if err := yaml.Unmarshal(yamlDoc, dep); err != nil {
		return err
	}
	depSpec := olmInstall.StrategyDeploymentSpec{
		Name: dep.ObjectMeta.Name,
		Spec: dep.Spec,
	}
	store.installStrategy.DeploymentSpecs = append(store.installStrategy.DeploymentSpecs, depSpec)

	return nil
}

func (u *CSVInstallStrategyUpdate) Apply(csv *olmApi.ClusterServiceVersion) (err error) {
	// Get install strategy from csv. Default to a deployment strategy if none found.
	var strat olmInstall.Strategy
	if csv.Spec.InstallStrategy.StrategyName == "" {
		csv.Spec.InstallStrategy.StrategyName = olmInstall.InstallStrategyNameDeployment
		strat = &olmInstall.StrategyDetailsDeployment{}
	} else {
		var resolver *olmInstall.StrategyResolver
		strat, err = resolver.UnmarshalStrategy(csv.Spec.InstallStrategy)
		if err != nil {
			return err
		}
	}

	switch s := strat.(type) {
	case *olmInstall.StrategyDetailsDeployment:
		// Update permissions and deployments.
		u.updatePermissions(s)
		u.updateClusterPermissions(s)
		u.updateDeploymentSpecs(s)
	default:
		return fmt.Errorf("install strategy (%v) of unknown type", strat)
	}

	// Re-serialize permissions into csv strategy.
	updatedStrat, err := json.Marshal(strat)
	if err != nil {
		return err
	}
	csv.Spec.InstallStrategy.StrategySpecRaw = updatedStrat

	return nil
}

func (u *CSVInstallStrategyUpdate) updatePermissions(strat *olmInstall.StrategyDetailsDeployment) {
	if len(u.Permissions) != 0 {
		strat.Permissions = u.Permissions
	}
}

func (u *CSVInstallStrategyUpdate) updateClusterPermissions(strat *olmInstall.StrategyDetailsDeployment) {
	if len(u.ClusterPermissions) != 0 {
		strat.ClusterPermissions = u.ClusterPermissions
	}
}

func (u *CSVInstallStrategyUpdate) updateDeploymentSpecs(strat *olmInstall.StrategyDetailsDeployment) {
	if len(u.DeploymentSpecs) != 0 {
		strat.DeploymentSpecs = u.DeploymentSpecs
	}
}

type CSVCustomResourceDefinitionsUpdate struct {
	*olmApi.CustomResourceDefinitions
}

func (store *updaterStore) AddOwnedCRD(yamlDoc []byte) error {
	crdDesc, err := parseCRDDescriptionFromYAML(yamlDoc)
	if err == nil {
		store.crdUpdate.Owned = append(store.crdUpdate.Owned, *crdDesc)
	}
	return err
}

func (store *updaterStore) AddRequiredCRD(yamlDoc []byte) error {
	crdDesc, err := parseCRDDescriptionFromYAML(yamlDoc)
	if err == nil {
		store.crdUpdate.Required = append(store.crdUpdate.Required, *crdDesc)
	}
	return err
}

func parseCRDDescriptionFromYAML(yamlDoc []byte) (*olmApi.CRDDescription, error) {
	crd := &apiextv1beta1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(yamlDoc, crd); err != nil {
		return nil, err
	}
	return &olmApi.CRDDescription{
		Name:    crd.ObjectMeta.Name,
		Version: crd.Spec.Version,
		Kind:    crd.Spec.Names.Kind,
	}, nil
}

// Apply updates all CRDDescriptions with any user-defined data in csv's
// CRDDescriptions.
func (u *CSVCustomResourceDefinitionsUpdate) Apply(csv *olmApi.ClusterServiceVersion) error {
	crdDescSet := make(map[string]*olmApi.CRDDescription)
	for _, desc := range u.Owned {
		crdDescSet[desc.Name] = &desc
	}
	for _, desc := range u.Required {
		crdDescSet[desc.Name] = &desc
	}
	for _, csvDesc := range csv.GetAllCRDDescriptions() {
		if uDesc, ok := crdDescSet[csvDesc.Name]; ok {
			uDesc.DisplayName = csvDesc.DisplayName
			uDesc.Description = csvDesc.Description
			uDesc.ActionDescriptor = csvDesc.ActionDescriptor
			uDesc.SpecDescriptors = csvDesc.SpecDescriptors
			uDesc.StatusDescriptors = csvDesc.StatusDescriptors
			uDesc.Resources = csvDesc.Resources
		}
	}
	csv.Spec.CustomResourceDefinitions = *u.CustomResourceDefinitions
	return nil
}
