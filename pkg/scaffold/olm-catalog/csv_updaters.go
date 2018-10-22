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
	"log"
	"regexp"
	"sync"

	"github.com/ghodss/yaml"
	olmApi "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olmInstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

type CSVUpdater interface {
	Apply(*olmApi.ClusterServiceVersion) error
}

type CSVUpdateSet struct {
	Updaters []CSVUpdater
}

func (s *CSVUpdateSet) Populate() {
	if localUpdater != nil {
		s.Updaters = append(s.Updaters, localUpdater.InstallStrategy)
		s.Updaters = append(s.Updaters, localUpdater.CustomResourceDefinitions)
	}
}

func (s *CSVUpdateSet) ApplyAll(csv *olmApi.ClusterServiceVersion) error {
	for _, updater := range s.Updaters {
		if err := updater.Apply(csv); err != nil {
			return err
		}
	}
	return nil
}

// TODO: allow custom Kinds that should be interpreted as standard
// k8s Kinds required in CSV's
var updateDispTable = map[string]func([]byte) error{
	"Role":       AddRoleToCSVInstallStrategyUpdate,
	"Deployment": AddDeploymentSpecToCSVInstallStrategyUpdate,
	// TODO: determine whether 'owned' or 'required'
	"CustomResourceDefinition": AddOwnedCRDToCSVCustomResourceDefinitionsUpdate,
}

type localUpdaterFactory struct {
	InstallStrategy           *CSVInstallStrategyUpdate
	CustomResourceDefinitions *CSVCustomResourceDefinitionsUpdate
}

var once sync.Once
var localUpdater *localUpdaterFactory

func getLocalUpdaterFactory() *localUpdaterFactory {
	once.Do(func() {
		localUpdater = &localUpdaterFactory{}

		localInstallStrategyUpdate := &CSVInstallStrategyUpdate{
			&olmInstall.StrategyDetailsDeployment{},
		}
		localInstallStrategyUpdate.DeploymentSpecs = make([]olmInstall.StrategyDeploymentSpec, 0)
		localInstallStrategyUpdate.Permissions = make([]olmInstall.StrategyDeploymentPermissions, 0)
		localInstallStrategyUpdate.ClusterPermissions = make([]olmInstall.StrategyDeploymentPermissions, 0)
		localUpdater.InstallStrategy = localInstallStrategyUpdate

		localCustomResourceDefinitionsUpdate := &CSVCustomResourceDefinitionsUpdate{
			&olmApi.CustomResourceDefinitions{},
		}
		localCustomResourceDefinitionsUpdate.Owned = make([]olmApi.CRDDescription, 0)
		localCustomResourceDefinitionsUpdate.Required = make([]olmApi.CRDDescription, 0)
		localUpdater.CustomResourceDefinitions = localCustomResourceDefinitionsUpdate
	})
	return localUpdater
}

type CSVInstallStrategyUpdate struct {
	*olmInstall.StrategyDetailsDeployment
}

func getLocalInstallStrategyUpdate() *CSVInstallStrategyUpdate {
	factory := getLocalUpdaterFactory()
	return factory.InstallStrategy
}

func AddRoleToCSVInstallStrategyUpdate(yamlDoc []byte) error {
	localISUpdate := getLocalInstallStrategyUpdate()

	newRole := new(rbacv1.Role)
	if err := yaml.Unmarshal(yamlDoc, newRole); err != nil {
		log.Printf("AddRoleToCSVInstallStrategyUpdate Unmarshal: (%v)", err)
		return err
	}
	newPerm := olmInstall.StrategyDeploymentPermissions{
		ServiceAccountName: newRole.ObjectMeta.Name,
		Rules:              newRole.Rules,
	}
	localISUpdate.Permissions = append(localISUpdate.Permissions, newPerm)

	return nil
}

func AddDeploymentSpecToCSVInstallStrategyUpdate(yamlDoc []byte) error {
	localISUpdate := getLocalInstallStrategyUpdate()

	newDep := new(appsv1.Deployment)
	if err := yaml.Unmarshal(yamlDoc, newDep); err != nil {
		log.Printf("AddDeploymentSpecToCSVInstallStrategyUpdate Unmarshal: (%v)", err)
		return err
	}
	newDepSpec := olmInstall.StrategyDeploymentSpec{
		Name: newDep.ObjectMeta.Name,
		Spec: newDep.Spec,
	}
	localISUpdate.DeploymentSpecs = append(localISUpdate.DeploymentSpecs, newDepSpec)

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
			log.Printf("[%T] UnmarshalStrategy: (%v)", *us, err)
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
		log.Printf("[%T] install strategy (%v) uf unknown type", *us, strat)
		return fmt.Errorf("install strategy (%v) of unknown type", strat)
	}

	// Re-serialize permissions into csv strategy.
	updatedStrat, err := json.Marshal(strat)
	if err != nil {
		log.Printf("[%T] Marshal: (%v)", *us, err)
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

func getLocalCustomResourceDefinitionsUpdate() *CSVCustomResourceDefinitionsUpdate {
	factory := getLocalUpdaterFactory()
	return factory.CustomResourceDefinitions
}

func AddOwnedCRDToCSVCustomResourceDefinitionsUpdate(yamlDoc []byte) error {
	localCRDsUpdate := getLocalCustomResourceDefinitionsUpdate()
	newCRDDesc, err := parseCRDDescriptionFromYAML(yamlDoc)
	if err == nil {
		localCRDsUpdate.Owned = append(localCRDsUpdate.Owned, *newCRDDesc)
	}
	return err
}

func AddRequiredCRDToCSVCustomResourceDefinitionsUpdate(yamlDoc []byte) error {
	localCRDsUpdate := getLocalCustomResourceDefinitionsUpdate()
	newCRDDesc, err := parseCRDDescriptionFromYAML(yamlDoc)
	if err == nil {
		localCRDsUpdate.Required = append(localCRDsUpdate.Required, *newCRDDesc)
	}
	return err
}

func parseCRDDescriptionFromYAML(yamlDoc []byte) (*olmApi.CRDDescription, error) {
	newCRD := new(apiextv1beta1.CustomResourceDefinition)
	if err := yaml.Unmarshal(yamlDoc, newCRD); err != nil {
		log.Printf("parseCRDDescriptionFromYAML Unmarshal: (%v)", err)
		return nil, err
	}

	crdDesc := &olmApi.CRDDescription{
		Name:    newCRD.ObjectMeta.Name,
		Version: newCRD.Spec.Version,
		Kind:    newCRD.Spec.Names.Kind,
		// Resources         []olmApi.CRDResourceReference	// Not sure where this is can be found
		// StatusDescriptors []olmApi.StatusDescriptor    	// Not sure where this is can be found
		// SpecDescriptors   []olmApi.SpecDescriptor      	// Not sure where this is can be found
		// ActionDescriptor  []olmApi.ActionDescriptor			// Not sure where this is can be found
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
		}
	}
}

// setCRDDescriptionIfExist sets dstCRDDesc's Description field only if a
// viable metadata annotation is found.
func setCRDDescriptionIfExist(dstCRDDesc *olmApi.CRDDescription, srcCRD *apiextv1beta1.CustomResourceDefinition) {
	for k, v := range srcCRD.ObjectMeta.GetAnnotations() {
		if descRegexp.MatchString(k) {
			dstCRDDesc.Description = v
		}
	}
}

func (us *CSVCustomResourceDefinitionsUpdate) Apply(csv *olmApi.ClusterServiceVersion) error {
	// Update all CRD's.
	csv.Spec.CustomResourceDefinitions = *us.CustomResourceDefinitions
	return nil
}
