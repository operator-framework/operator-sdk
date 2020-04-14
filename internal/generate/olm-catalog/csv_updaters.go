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

package olmcatalog

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"sort"
	"strings"

	"github.com/operator-framework/operator-registry/pkg/registry"
	"github.com/operator-framework/operator-sdk/internal/generate/olm-catalog/descriptor"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/ghodss/yaml"
	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
)

// manifestCollection holds a collection of all manifests relevant to CSV updates.
type manifestCollection struct {
	Roles                     []rbacv1.Role
	ClusterRoles              []rbacv1.ClusterRole
	Deployments               []appsv1.Deployment
	CustomResourceDefinitions []apiextv1beta1.CustomResourceDefinition
	CustomResources           []unstructured.Unstructured
	Others                    []unstructured.Unstructured
}

// addRoles assumes add manifest data in rawManifests are Roles and adds them
// to the collection.
func (c *manifestCollection) addRoles(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		role := rbacv1.Role{}
		if err := yaml.Unmarshal(rawManifest, &role); err != nil {
			return fmt.Errorf("error adding Role to manifest collection: %v", err)
		}
		c.Roles = append(c.Roles, role)
	}
	return nil
}

// addClusterRoles assumes add manifest data in rawManifests are ClusterRoles
// and adds them to the collection.
func (c *manifestCollection) addClusterRoles(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		role := rbacv1.ClusterRole{}
		if err := yaml.Unmarshal(rawManifest, &role); err != nil {
			return fmt.Errorf("error adding ClusterRole to manifest collection: %v", err)
		}
		c.ClusterRoles = append(c.ClusterRoles, role)
	}
	return nil
}

// addDeployments assumes add manifest data in rawManifests are Deployments
// and adds them to the collection.
func (c *manifestCollection) addDeployments(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		dep := appsv1.Deployment{}
		if err := yaml.Unmarshal(rawManifest, &dep); err != nil {
			return fmt.Errorf("error adding Deployment to manifest collection: %v", err)
		}
		c.Deployments = append(c.Deployments, dep)
	}
	return nil
}

// addOthers assumes add manifest data in rawManifests are able to be
// unmarshalled into an Unstructured object and adds them to the collection.
func (c *manifestCollection) addOthers(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		u := unstructured.Unstructured{}
		if err := yaml.Unmarshal(rawManifest, &u); err != nil {
			return fmt.Errorf("error adding manifest collection: %v", err)
		}
		c.Others = append(c.Others, u)
	}
	return nil
}

// filter applies filtering rules to certain manifest types in a collection.
func (c *manifestCollection) filter() {
	c.filterCustomResources()
}

// filterCustomResources filters "other" objects, which contain likely
// Custom Resources corresponding to a CustomResourceDefinition, by GVK.
func (c *manifestCollection) filterCustomResources() {
	crdGVKSet := make(map[schema.GroupVersionKind]struct{})
	for _, crd := range c.CustomResourceDefinitions {
		for _, version := range crd.Spec.Versions {
			gvk := schema.GroupVersionKind{
				Group:   crd.Spec.Group,
				Version: version.Name,
				Kind:    crd.Spec.Names.Kind,
			}
			crdGVKSet[gvk] = struct{}{}
		}
	}

	customResources := []unstructured.Unstructured{}
	for _, other := range c.Others {
		if _, gvkMatches := crdGVKSet[other.GroupVersionKind()]; gvkMatches {
			customResources = append(customResources, other)
		}
	}
	c.CustomResources = customResources
}

// apply applies the manifests in the collection to csv.
func (c manifestCollection) apply(csv *operatorsv1alpha1.ClusterServiceVersion) error {
	strategy := getCSVInstallStrategy(csv)
	switch strategy.StrategyName {
	case operatorsv1alpha1.InstallStrategyNameDeployment:
		c.applyRoles(&strategy.StrategySpec)
		c.applyClusterRoles(&strategy.StrategySpec)
		c.applyDeployments(&strategy.StrategySpec)
	}
	csv.Spec.InstallStrategy = strategy

	c.applyCustomResourceDefinitions(csv)
	if err := c.applyCustomResources(csv); err != nil {
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
func (c manifestCollection) applyRoles(strategy *operatorsv1alpha1.StrategyDetailsDeployment) {
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
func (c manifestCollection) applyClusterRoles(strategy *operatorsv1alpha1.StrategyDetailsDeployment) {
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
func (c manifestCollection) applyDeployments(strategy *operatorsv1alpha1.StrategyDetailsDeployment) {
	depSpecs := []operatorsv1alpha1.StrategyDeploymentSpec{}
	for _, dep := range c.Deployments {
		setWatchNamespacesEnv(&dep)
		// Make sure "olm.targetNamespaces" is referenced somewhere in dep,
		// and emit a warning of not.
		if !depHasOLMNamespaces(dep) {
			log.Warnf(`No WATCH_NAMESPACE environment variable nor reference to "%s"`+
				` detected in operator Deployment. For OLM compatibility, your operator`+
				` MUST watch namespaces defined in "%s"`, olmTNMeta, olmTNMeta)
		}
		depSpecs = append(depSpecs, operatorsv1alpha1.StrategyDeploymentSpec{
			Name: dep.GetName(),
			Spec: dep.Spec,
		})
	}
	strategy.DeploymentSpecs = depSpecs
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
	return bytes.Contains(b, []byte(olmTNMeta))
}

// applyCustomResourceDefinitions updates csv's customresourcedefinitions.owned
// with CustomResourceDefinitions in the collection.
// customresourcedefinitions.required are left as-is, since they are
// manually-defined values.
func (c manifestCollection) applyCustomResourceDefinitions(csv *operatorsv1alpha1.ClusterServiceVersion) {
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

// updateDescriptions parses APIs in apisDir for code and annotations that
// can build a verbose crdDescription and updates existing crdDescriptions in
// csv. If no code/annotations are found, the crdDescription is appended as-is.
func updateDescriptions(csv *operatorsv1alpha1.ClusterServiceVersion, apisDir string) error {
	updatedDescriptions := []operatorsv1alpha1.CRDDescription{}
	for _, currDescription := range csv.Spec.CustomResourceDefinitions.Owned {
		group := currDescription.Name
		if split := strings.Split(currDescription.Name, "."); len(split) > 1 {
			group = strings.Join(split[1:], ".")
		}
		// Parse CRD descriptors from source code comments and annotations.
		gvk := schema.GroupVersionKind{
			Group:   group,
			Version: currDescription.Version,
			Kind:    currDescription.Kind,
		}
		newDescription, err := descriptor.GetCRDDescriptionForGVK(apisDir, gvk)
		if err != nil {
			if goerrors.Is(err, descriptor.ErrAPIDirNotExist) {
				log.Debugf("Directory for API %s does not exist. Skipping CSV annotation parsing for API.", gvk)
			} else if goerrors.Is(err, descriptor.ErrAPITypeNotFound) {
				log.Debugf("No kind type found for API %s. Skipping CSV annotation parsing for API.", gvk)
			} else {
				// TODO: Should we ignore all CSV annotation parsing errors and simply log the error
				// like we do for the above cases.
				return fmt.Errorf("failed to set CRD descriptors for %s: %v", gvk, err)
			}
			// Keep the existing description and don't update on error
			updatedDescriptions = append(updatedDescriptions, currDescription)
		} else {
			// Replace the existing description with the newly parsed one
			newDescription.Name = currDescription.Name
			updatedDescriptions = append(updatedDescriptions, newDescription)
		}
	}
	csv.Spec.CustomResourceDefinitions.Owned = updatedDescriptions
	return nil
}

// applyCustomResources updates csv's "alm-examples" annotation with the
// Custom Resources in the collection.
func (c manifestCollection) applyCustomResources(csv *operatorsv1alpha1.ClusterServiceVersion) error {
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

// deduplicate removes duplicate objects from the collection, since we are
// collecting an arbitrary list of manifests.
func (c *manifestCollection) deduplicate() error {
	hashes := make(map[string]struct{})

	roles := []rbacv1.Role{}
	for _, role := range c.Roles {
		hasHash, err := addToHashes(&role, hashes)
		if err != nil {
			return err
		}
		if !hasHash {
			roles = append(roles, role)
		}
	}
	c.Roles = roles

	clusterRoles := []rbacv1.ClusterRole{}
	for _, clusterRole := range c.ClusterRoles {
		hasHash, err := addToHashes(&clusterRole, hashes)
		if err != nil {
			return err
		}
		if !hasHash {
			clusterRoles = append(clusterRoles, clusterRole)
		}
	}
	c.ClusterRoles = clusterRoles

	deps := []appsv1.Deployment{}
	for _, dep := range c.Deployments {
		hasHash, err := addToHashes(&dep, hashes)
		if err != nil {
			return err
		}
		if !hasHash {
			deps = append(deps, dep)
		}
	}
	c.Deployments = deps

	crds := []apiextv1beta1.CustomResourceDefinition{}
	for _, crd := range c.CustomResourceDefinitions {
		hasHash, err := addToHashes(&crd, hashes)
		if err != nil {
			return err
		}
		if !hasHash {
			crds = append(crds, crd)
		}
	}
	c.CustomResourceDefinitions = crds

	crs := []unstructured.Unstructured{}
	for _, cr := range c.CustomResources {
		b, err := cr.MarshalJSON()
		if err != nil {
			return err
		}
		hash := hashContents(b)
		if _, hasHash := hashes[hash]; !hasHash {
			crs = append(crs, cr)
			hashes[hash] = struct{}{}
		}
	}
	c.CustomResources = crs

	return nil
}

// marshaller is an interface used to generalize hashing for deduplication.
type marshaller interface {
	Marshal() ([]byte, error)
}

// addToHashes calls m.Marshal(), hashes the returned bytes, and adds the
// hash to hashes if it does not exist. addToHashes returns true if m's hash
// was not in hashes.
func addToHashes(m marshaller, hashes map[string]struct{}) (bool, error) {
	b, err := m.Marshal()
	if err != nil {
		return false, err
	}
	hash := hashContents(b)
	_, hasHash := hashes[hash]
	if !hasHash {
		hashes[hash] = struct{}{}
	}
	return hasHash, nil
}

// hashContents creates a sha256 md5 digest of b's bytes.
func hashContents(b []byte) string {
	h := sha256.New()
	_, _ = h.Write(b)
	return string(h.Sum(nil))
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
