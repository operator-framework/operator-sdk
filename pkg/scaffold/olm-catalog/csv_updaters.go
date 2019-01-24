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
	"regexp"

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
	newRole := &rbacv1.Role{}
	if err := yaml.Unmarshal(yamlDoc, newRole); err != nil {
		return err
	}
	newPerm := olmInstall.StrategyDeploymentPermissions{
		ServiceAccountName: newRole.ObjectMeta.Name,
		Rules:              newRole.Rules,
	}
	store.installStrategy.Permissions = append(store.installStrategy.Permissions, newPerm)

	return nil
}

func (store *updaterStore) AddClusterRole(yamlDoc []byte) error {
	newCRole := &rbacv1.ClusterRole{}
	if err := yaml.Unmarshal(yamlDoc, newCRole); err != nil {
		return err
	}
	newPerm := olmInstall.StrategyDeploymentPermissions{
		ServiceAccountName: newCRole.ObjectMeta.Name,
		Rules:              newCRole.Rules,
	}
	store.installStrategy.ClusterPermissions = append(store.installStrategy.ClusterPermissions, newPerm)

	return nil
}

func (store *updaterStore) AddDeploymentSpec(yamlDoc []byte) error {
	newDep := &appsv1.Deployment{}
	if err := yaml.Unmarshal(yamlDoc, newDep); err != nil {
		return err
	}
	newDepSpec := olmInstall.StrategyDeploymentSpec{
		Name: newDep.ObjectMeta.Name,
		Spec: newDep.Spec,
	}
	store.installStrategy.DeploymentSpecs = append(store.installStrategy.DeploymentSpecs, newDepSpec)

	return nil
}

func (us *CSVInstallStrategyUpdate) Apply(csv *olmApi.ClusterServiceVersion) (err error) {
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
		us.updatePermissions(s)
		us.updateClusterPermissions(s)
		us.updateDeploymentSpecs(s)
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

func (us *CSVInstallStrategyUpdate) updatePermissions(strat *olmInstall.StrategyDetailsDeployment) {
	if len(us.Permissions) != 0 {
		strat.Permissions = us.Permissions
	}
}

func (us *CSVInstallStrategyUpdate) updateClusterPermissions(strat *olmInstall.StrategyDetailsDeployment) {
	if len(us.ClusterPermissions) != 0 {
		strat.ClusterPermissions = us.ClusterPermissions
	}
}

func (us *CSVInstallStrategyUpdate) updateDeploymentSpecs(strat *olmInstall.StrategyDetailsDeployment) {
	if len(us.DeploymentSpecs) != 0 {
		strat.DeploymentSpecs = us.DeploymentSpecs
	}
}

type CSVCustomResourceDefinitionsUpdate struct {
	*olmApi.CustomResourceDefinitions
}

func (store *updaterStore) AddOwnedCRD(yamlDoc []byte) error {
	newCRDDesc, err := parseCRDDescriptionFromYAML(yamlDoc)
	if err == nil {
		store.crdUpdate.Owned = append(store.crdUpdate.Owned, *newCRDDesc)
	}
	return err
}

func (store *updaterStore) AddRequiredCRD(yamlDoc []byte) error {
	newCRDDesc, err := parseCRDDescriptionFromYAML(yamlDoc)
	if err == nil {
		store.crdUpdate.Required = append(store.crdUpdate.Required, *newCRDDesc)
	}
	return err
}

func parseCRDDescriptionFromYAML(yamlDoc []byte) (*olmApi.CRDDescription, error) {
	newCRD := &apiextv1beta1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(yamlDoc, newCRD); err != nil {
		return nil, err
	}

	crdDesc := &olmApi.CRDDescription{
		Name:    newCRD.ObjectMeta.Name,
		Version: newCRD.Spec.Version,
		Kind:    newCRD.Spec.Names.Kind,
	}
	setCRDDisplayNameIfExist(crdDesc, newCRD)
	setCRDDescriptionIfExist(crdDesc, newCRD)

	return crdDesc, nil
}

// Annotations for display name and description in CRD metadata should have
// keys containing strings that match the following patterns if users want
// CSV's to correctly describe and display their CRD's.
var (
	dnRegexp   = regexp.MustCompile(".*[dD]isplay[nN]ame.*")
	descRegexp = regexp.MustCompile(".*[dD]escription.*")
)

// setCRDDisplayNameIfExist sets dstCRDDesc's DisplayName field only if a
// viable metadata annotation is found.
func setCRDDisplayNameIfExist(dstCRDDesc *olmApi.CRDDescription, srcCRD *apiextv1beta1.CustomResourceDefinition) {
	for k, v := range srcCRD.ObjectMeta.GetAnnotations() {
		if dnRegexp.MatchString(k) {
			dstCRDDesc.DisplayName = v
			break
		}
	}
}

// setCRDDescriptionIfExist sets dstCRDDesc's Description field only if a
// viable metadata annotation is found.
func setCRDDescriptionIfExist(dstCRDDesc *olmApi.CRDDescription, srcCRD *apiextv1beta1.CustomResourceDefinition) {
	for k, v := range srcCRD.ObjectMeta.GetAnnotations() {
		if descRegexp.MatchString(k) {
			dstCRDDesc.Description = v
			break
		}
	}
}

func (us *CSVCustomResourceDefinitionsUpdate) Apply(csv *olmApi.ClusterServiceVersion) error {
	// Update all CRD descriptions, and include any user-written information in
	// the new CRD descriptions.
	crdDescSet := make(map[string]*olmApi.CRDDescription)
	for _, desc := range us.Owned {
		crdDescSet[desc.Name] = &desc
	}
	for _, desc := range us.Required {
		crdDescSet[desc.Name] = &desc
	}
	for _, crd := range csv.GetAllCRDDescriptions() {
		if desc, ok := crdDescSet[crd.Name]; ok {
			desc.ActionDescriptor = crd.ActionDescriptor
			desc.SpecDescriptors = crd.SpecDescriptors
			desc.StatusDescriptors = crd.StatusDescriptors
			desc.Resources = crd.Resources
		}
	}
	csv.Spec.CustomResourceDefinitions = *us.CustomResourceDefinitions
	return nil
}
