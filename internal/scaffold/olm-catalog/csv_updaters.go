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
	goerrors "errors"
	"sort"

	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/scaffold/olm-catalog/descriptor"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/pkg/errors"

	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olminstall "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/install"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
)

// csvUpdater is an interface for any data that can be in a CSV, which will be
// set to the corresponding field on apply().
type csvUpdater interface {
	// apply applies a data update to a CSV argument.
	apply(*olmapiv1alpha1.ClusterServiceVersion) error
}

// Get install strategy from csv.
func getCSVInstallStrategy(csv *olmapiv1alpha1.ClusterServiceVersion) (strategy olminstall.Strategy, err error) {
	// Default to a deployment strategy if none found.
	if len(csv.Spec.InstallStrategy.StrategySpecRaw) == 0 || csv.Spec.InstallStrategy.StrategyName == "" {
		csv.Spec.InstallStrategy.StrategyName = olminstall.InstallStrategyNameDeployment
		return &olminstall.StrategyDetailsDeployment{}, nil
	}
	return (&olminstall.StrategyResolver{}).UnmarshalStrategy(csv.Spec.InstallStrategy)
}

// Set csv's spec.install to strategy.
func setCSVInstallStrategy(csv *olmapiv1alpha1.ClusterServiceVersion, strategy olminstall.Strategy) error {
	sb, err := json.Marshal(strategy)
	if err != nil {
		return err
	}
	csv.Spec.InstallStrategy.StrategySpecRaw = json.RawMessage(sb)
	return nil
}

type roles [][]byte

var _ csvUpdater = roles{}

func (us roles) apply(csv *olmapiv1alpha1.ClusterServiceVersion) (err error) {
	strategy, err := getCSVInstallStrategy(csv)
	if err != nil {
		return err
	}
	switch s := strategy.(type) {
	case *olminstall.StrategyDetailsDeployment:
		perms := []olminstall.StrategyDeploymentPermissions{}
		for _, u := range us {
			role := rbacv1.Role{}
			if err := yaml.Unmarshal(u, &role); err != nil {
				return err
			}
			perms = append(perms, olminstall.StrategyDeploymentPermissions{
				ServiceAccountName: role.GetName(),
				Rules:              role.Rules,
			})
		}
		s.Permissions = perms
	}
	if err = setCSVInstallStrategy(csv, strategy); err != nil {
		return err
	}
	return nil
}

type clusterRoles [][]byte

var _ csvUpdater = clusterRoles{}

func (us clusterRoles) apply(csv *olmapiv1alpha1.ClusterServiceVersion) (err error) {
	strategy, err := getCSVInstallStrategy(csv)
	if err != nil {
		return err
	}
	switch s := strategy.(type) {
	case *olminstall.StrategyDetailsDeployment:
		perms := []olminstall.StrategyDeploymentPermissions{}
		for _, u := range us {
			clusterRole := rbacv1.ClusterRole{}
			if err := yaml.Unmarshal(u, &clusterRole); err != nil {
				return err
			}
			perms = append(perms, olminstall.StrategyDeploymentPermissions{
				ServiceAccountName: clusterRole.GetName(),
				Rules:              clusterRole.Rules,
			})
		}
		s.ClusterPermissions = perms
	}
	if err = setCSVInstallStrategy(csv, strategy); err != nil {
		return err
	}
	return nil
}

type deployments [][]byte

var _ csvUpdater = deployments{}

func (us deployments) apply(csv *olmapiv1alpha1.ClusterServiceVersion) (err error) {
	strategy, err := getCSVInstallStrategy(csv)
	if err != nil {
		return err
	}
	switch s := strategy.(type) {
	case *olminstall.StrategyDetailsDeployment:
		depSpecs := []olminstall.StrategyDeploymentSpec{}
		for _, u := range us {
			dep := appsv1.Deployment{}
			if err := yaml.Unmarshal(u, &dep); err != nil {
				return err
			}
			setWatchNamespacesEnv(&dep)
			// Make sure "olm.targetNamespaces" is referenced somewhere in dep,
			// and emit a warning of not.
			if !depHasOLMNamespaces(dep) {
				log.Warnf(`No WATCH_NAMESPACE environment variable nor reference to "%s"`+
					` detected in operator Deployment. For OLM compatibility, your operator`+
					` MUST watch namespaces defined in "%s"`, olmTNMeta, olmTNMeta)
			}
			depSpecs = append(depSpecs, olminstall.StrategyDeploymentSpec{
				Name: dep.GetName(),
				Spec: dep.Spec,
			})
		}
		s.DeploymentSpecs = depSpecs
	}
	if err = setCSVInstallStrategy(csv, strategy); err != nil {
		return err
	}
	return nil
}

const olmTNMeta = "metadata.annotations['olm.targetNamespaces']"

// setWatchNamespacesEnv sets WATCH_NAMESPACE to olmTNString in dep if
// WATCH_NAMESPACE exists in a pod spec container's env list.
func setWatchNamespacesEnv(dep *appsv1.Deployment) {
	envVar := newEnvVar(k8sutil.WatchNamespaceEnvVar, olmTNMeta)
	overwriteContainerEnvVar(dep, k8sutil.WatchNamespaceEnvVar, envVar)
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
func depHasOLMNamespaces(dep appsv1.Deployment) bool {
	b, err := dep.Spec.Template.Marshal()
	if err != nil {
		// Something is wrong with the deployment manifest, not with CLI inputs.
		log.Fatalf("Marshal Deployment spec: %v", err)
	}
	return bytes.Index(b, []byte(olmTNMeta)) != -1
}

type descSorter []olmapiv1alpha1.CRDDescription

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

type crds [][]byte

var _ csvUpdater = crds{}

// apply updates csv's "owned" CRDDescriptions. "required" CRDDescriptions are
// left as-is, since they are user-defined values.
// apply will only make a new spec.customresourcedefinitions.owned element for
// a type if an annotation is present on that type's declaration.
func (us crds) apply(csv *olmapiv1alpha1.ClusterServiceVersion) error {
	ownedCRDs := []olmapiv1alpha1.CRDDescription{}
	for _, u := range us {
		crd := apiextv1beta1.CustomResourceDefinition{}
		if err := yaml.Unmarshal(u, &crd); err != nil {
			return err
		}
		for _, ver := range crd.Spec.Versions {
			// Parse CRD descriptors from source code comments and annotations.
			gvk := schema.GroupVersionKind{
				Group:   crd.Spec.Group,
				Version: ver.Name,
				Kind:    crd.Spec.Names.Kind,
			}
			newCRDDesc, err := descriptor.GetCRDDescriptionForGVK(scaffold.ApisDir, gvk)
			if err != nil {
				if goerrors.Is(err, descriptor.ErrAPIDirNotExist) {
					log.Infof("Directory for API %s does not exist. Skipping CSV annotation parsing for API.", gvk)
				} else if goerrors.Is(err, descriptor.ErrAPITypeNotFound) {
					log.Infof("No kind type found for API %s. Skipping CSV annotation parsing for API.", gvk)
				} else {
					return errors.Wrapf(err, "failed to set CRD descriptors for %s", gvk)
				}
				continue
			}
			// Only set the name if no error was returned.
			newCRDDesc.Name = crd.GetName()
			ownedCRDs = append(ownedCRDs, newCRDDesc)
		}
	}
	csv.Spec.CustomResourceDefinitions.Owned = ownedCRDs
	sort.Sort(descSorter(csv.Spec.CustomResourceDefinitions.Owned))
	sort.Sort(descSorter(csv.Spec.CustomResourceDefinitions.Required))
	return nil
}

type crs [][]byte

var _ csvUpdater = crs{}

func (us crs) apply(csv *olmapiv1alpha1.ClusterServiceVersion) error {
	examples := []json.RawMessage{}
	for _, u := range us {
		crBytes, err := yaml.YAMLToJSON(u)
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
