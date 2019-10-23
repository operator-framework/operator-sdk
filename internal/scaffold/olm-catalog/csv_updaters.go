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
	"bytes"
	"encoding/json"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olminstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CSVUpdater is an interface for any data that can be in a CSV, which will be
// set to the corresponding field on Apply().
type CSVUpdater interface {
	// Apply applies a data update to a CSV argument.
	Apply(*olmapiv1alpha1.ClusterServiceVersion) error
}

type updaterStore struct {
	installStrategy *InstallStrategyUpdate
	crds            *CustomResourceDefinitionsUpdate
	almExamples     *ALMExamplesUpdate
}

func NewUpdaterStore() *updaterStore {
	return &updaterStore{
		installStrategy: &InstallStrategyUpdate{
			&olminstall.StrategyDetailsDeployment{},
		},
		crds: &CustomResourceDefinitionsUpdate{
			&olmapiv1alpha1.CustomResourceDefinitions{},
			make(map[string]struct{}),
		},
		almExamples: &ALMExamplesUpdate{},
	}
}

// Apply iteratively calls each stored CSVUpdater's Apply() method.
func (s *updaterStore) Apply(csv *olmapiv1alpha1.ClusterServiceVersion) error {
	updaters := []CSVUpdater{s.installStrategy, s.crds, s.almExamples}
	for _, updater := range updaters {
		if err := updater.Apply(csv); err != nil {
			return err
		}
	}
	return nil
}

func (s *updaterStore) AddToUpdater(yamlSpec []byte, kind string) (found bool, err error) {
	found = true
	switch kind {
	case "Role":
		err = s.AddRole(yamlSpec)
	case "ClusterRole":
		err = s.AddClusterRole(yamlSpec)
	case "Deployment":
		err = s.AddDeploymentSpec(yamlSpec)
	case "CustomResourceDefinition":
		// All CRD's present will be 'owned'.
		err = s.AddOwnedCRD(yamlSpec)
	default:
		found = false
	}
	return found, err
}

type InstallStrategyUpdate struct {
	*olminstall.StrategyDetailsDeployment
}

func (store *updaterStore) AddRole(yamlDoc []byte) error {
	role := &rbacv1.Role{}
	if err := yaml.Unmarshal(yamlDoc, role); err != nil {
		return err
	}
	perm := olminstall.StrategyDeploymentPermissions{
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
	perm := olminstall.StrategyDeploymentPermissions{
		ServiceAccountName: clusterRole.ObjectMeta.Name,
		Rules:              clusterRole.Rules,
	}
	store.installStrategy.ClusterPermissions = append(store.installStrategy.ClusterPermissions, perm)

	return nil
}

const olmTNMeta = "metadata.annotations['olm.targetNamespaces']"

func (store *updaterStore) AddDeploymentSpec(yamlDoc []byte) error {
	dep := &appsv1.Deployment{}
	if err := yaml.Unmarshal(yamlDoc, dep); err != nil {
		return err
	}

	setWatchNamespacesEnv(dep)
	// Make sure "olm.targetNamespaces" is referenced somewhere in dep,
	// and emit a warning of not.
	if !depHasOLMNamespaces(dep) {
		log.Warnf(`No WATCH_NAMESPACE environment variable nor reference to "%s"`+
			` detected in operator Deployment. For OLM compatibility, your operator`+
			` MUST watch namespaces defined in "%s"`, olmTNMeta, olmTNMeta)
	}

	depSpec := olminstall.StrategyDeploymentSpec{
		Name: dep.ObjectMeta.Name,
		Spec: dep.Spec,
	}
	store.installStrategy.DeploymentSpecs = append(store.installStrategy.DeploymentSpecs, depSpec)

	return nil
}

// setWatchNamespacesEnv sets WATCH_NAMESPACE to olmTNString in dep if
// WATCH_NAMESPACE exists in a pod spec container's env list.
func setWatchNamespacesEnv(dep *appsv1.Deployment) {
	overwriteContainerEnvVar(dep, k8sutil.WatchNamespaceEnvVar, newEnvVar(k8sutil.WatchNamespaceEnvVar, olmTNMeta))
}

func overwriteContainerEnvVar(dep *appsv1.Deployment, name string, ev corev1.EnvVar) {
	for _, c := range dep.Spec.Template.Spec.Containers {
		for i := 0; i < len(c.Env); i++ {
			if c.Env[i].Name == name {
				c.Env[i] = ev
			}
		}
	}
}

func newEnvVar(name, fieldPath string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: fieldPath,
			},
		},
	}
}

// OLM places the set of target namespaces for the operator in
// "metadata.annotations['olm.targetNamespaces']". This value should be
// referenced in either:
//	- The Deployment's pod spec WATCH_NAMESPACE env variable.
//	- Some other Deployment pod spec field.
func depHasOLMNamespaces(dep *appsv1.Deployment) bool {
	b, err := dep.Spec.Template.Marshal()
	if err != nil {
		// Something is wrong with the deployment manifest, not with CLI inputs.
		log.Fatalf("Marshal Deployment spec: %v", err)
	}
	return bytes.Index(b, []byte(olmTNMeta)) != -1
}

func (u *InstallStrategyUpdate) Apply(csv *olmapiv1alpha1.ClusterServiceVersion) (err error) {
	// Get install strategy from csv. Default to a deployment strategy if none found.
	var strat olminstall.Strategy
	if csv.Spec.InstallStrategy.StrategyName == "" {
		csv.Spec.InstallStrategy.StrategyName = olminstall.InstallStrategyNameDeployment
		strat = &olminstall.StrategyDetailsDeployment{}
	} else {
		var resolver *olminstall.StrategyResolver
		strat, err = resolver.UnmarshalStrategy(csv.Spec.InstallStrategy)
		if err != nil {
			return err
		}
	}

	switch s := strat.(type) {
	case *olminstall.StrategyDetailsDeployment:
		// Update permissions and deployments.
		u.updatePermissions(s)
		u.updateClusterPermissions(s)
		u.updateDeploymentSpecs(s)
	default:
		return errors.Errorf("install strategy (%v) of unknown type", strat)
	}

	// Re-serialize permissions into csv strategy.
	updatedStrat, err := json.Marshal(strat)
	if err != nil {
		return err
	}
	csv.Spec.InstallStrategy.StrategySpecRaw = updatedStrat

	return nil
}

func (u *InstallStrategyUpdate) updatePermissions(strat *olminstall.StrategyDetailsDeployment) {
	if len(u.Permissions) != 0 {
		strat.Permissions = u.Permissions
	}
}

func (u *InstallStrategyUpdate) updateClusterPermissions(strat *olminstall.StrategyDetailsDeployment) {
	if len(u.ClusterPermissions) != 0 {
		strat.ClusterPermissions = u.ClusterPermissions
	}
}

func (u *InstallStrategyUpdate) updateDeploymentSpecs(strat *olminstall.StrategyDetailsDeployment) {
	if len(u.DeploymentSpecs) != 0 {
		strat.DeploymentSpecs = u.DeploymentSpecs
	}
}

type CustomResourceDefinitionsUpdate struct {
	*olmapiv1alpha1.CustomResourceDefinitions
	crIDs map[string]struct{}
}

func (store *updaterStore) AddOwnedCRD(yamlDoc []byte) error {
	crd := &apiextv1beta1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(yamlDoc, crd); err != nil {
		return err
	}
	versions, err := getCRDVersions(crd)
	if err != nil {
		return errors.Wrapf(err, "failed to get owned CRD %s versions", crd.GetName())
	}
	for _, ver := range versions {
		kind := crd.Spec.Names.Kind
		crdDesc := olmapiv1alpha1.CRDDescription{
			Name:    crd.ObjectMeta.Name,
			Version: ver,
			Kind:    kind,
		}
		store.crds.crIDs[crdDescID(crdDesc)] = struct{}{}
		store.crds.Owned = append(store.crds.Owned, crdDesc)
	}
	return nil
}

func getCRDVersions(crd *apiextv1beta1.CustomResourceDefinition) (versions []string, err error) {
	if len(crd.Spec.Versions) != 0 {
		for _, ver := range crd.Spec.Versions {
			// Only versions served by the API server are relevant to a CSV.
			if ver.Served {
				versions = append(versions, ver.Name)
			}
		}
	} else if crd.Spec.Version != "" {
		versions = append(versions, crd.Spec.Version)
	}
	if len(versions) == 0 {
		return nil, errors.Errorf("no versions in CRD %s", crd.GetName())
	}
	return versions, nil
}

// crdDescID produces an opaque, unique string identifying a CRDDescription.
func crdDescID(desc olmapiv1alpha1.CRDDescription) string {
	// Name should always be <lower kind>.<group>, so this is effectively a GVK.
	splitName := strings.Split(desc.Name, ".")
	return getGVKID(strings.Join(splitName[1:], "."), desc.Version, desc.Kind)
}

// gvkID produces an opaque, unique string identifying a GVK.
func gvkID(gvk schema.GroupVersionKind) string {
	return getGVKID(gvk.Group, gvk.Version, gvk.Kind)
}

func getGVKID(g, v, k string) string {
	return g + v + k
}

// Apply updates csv's "owned" CRDDescriptions. "required" CRDDescriptions are
// left as-is, since they are user-defined values.
// Apply will only make a new spec.customresourcedefinitions.owned element if
// the CRD key is not in spec.customresourcedefinitions.owned already.
func (u *CustomResourceDefinitionsUpdate) Apply(csv *olmapiv1alpha1.ClusterServiceVersion) error {
	set := make(map[string]olmapiv1alpha1.CRDDescription)
	for _, csvDesc := range csv.Spec.CustomResourceDefinitions.Owned {
		set[crdDescID(csvDesc)] = csvDesc
	}
	newDescs := []olmapiv1alpha1.CRDDescription{}
	for _, uDesc := range u.Owned {
		if csvDesc, ok := set[crdDescID(uDesc)]; !ok {
			newDescs = append(newDescs, uDesc)
		} else {
			newDescs = append(newDescs, csvDesc)
		}
	}
	csv.Spec.CustomResourceDefinitions.Owned = newDescs
	return nil
}

type ALMExamplesUpdate struct {
	crs []string
}

func (store *updaterStore) AddCR(yamlDoc []byte) error {
	if len(yamlDoc) == 0 {
		return nil
	}
	crBytes, err := yaml.YAMLToJSON(yamlDoc)
	if err != nil {
		return err
	}
	store.almExamples.crs = append(store.almExamples.crs, string(crBytes))
	return nil
}

func (u *ALMExamplesUpdate) Apply(csv *olmapiv1alpha1.ClusterServiceVersion) error {
	if len(u.crs) == 0 {
		return nil
	}
	if csv.GetAnnotations() == nil {
		csv.SetAnnotations(make(map[string]string))
	}
	buf := &bytes.Buffer{}
	buf.WriteString(`[`)
	for i, example := range u.crs {
		buf.WriteString(example)
		if i < len(u.crs)-1 {
			buf.WriteString(`,`)
		}
	}
	buf.WriteString(`]`)
	examplesJSON, err := prettyJSON(buf.Bytes())
	if err != nil {
		return err
	}
	csv.GetAnnotations()["alm-examples"] = examplesJSON
	return nil
}

// prettyJSON returns a JSON in a pretty format
func prettyJSON(b []byte) (string, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")
	return out.String(), err
}
