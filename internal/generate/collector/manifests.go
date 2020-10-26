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

package collector

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	scorecardv1alpha3 "github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	log "github.com/sirupsen/logrus"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

// Manifests holds a collector of all manifests relevant to CSV updates.
type Manifests struct {
	Roles                            []rbacv1.Role
	ClusterRoles                     []rbacv1.ClusterRole
	RoleBindings                     []rbacv1.RoleBinding
	ClusterRoleBindings              []rbacv1.ClusterRoleBinding
	Deployments                      []appsv1.Deployment
	ServiceAccounts                  []corev1.ServiceAccount
	Services                         []corev1.Service
	V1CustomResourceDefinitions      []apiextv1.CustomResourceDefinition
	V1beta1CustomResourceDefinitions []apiextv1beta1.CustomResourceDefinition
	ValidatingWebhooks               []admissionregv1.ValidatingWebhook
	MutatingWebhooks                 []admissionregv1.MutatingWebhook
	CustomResources                  []unstructured.Unstructured
	ScorecardConfig                  scorecardv1alpha3.Configuration

	Others []unstructured.Unstructured
}

var (
	roleGK                 = rbacv1.SchemeGroupVersion.WithKind("Role").GroupKind()
	clusterRoleGK          = rbacv1.SchemeGroupVersion.WithKind("ClusterRole").GroupKind()
	roleBindingGK          = rbacv1.SchemeGroupVersion.WithKind("RoleBinding").GroupKind()
	clusterRoleBindingGK   = rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding").GroupKind()
	serviceAccountGK       = corev1.SchemeGroupVersion.WithKind("ServiceAccount").GroupKind()
	serviceGK              = corev1.SchemeGroupVersion.WithKind("Service").GroupKind()
	deploymentGK           = appsv1.SchemeGroupVersion.WithKind("Deployment").GroupKind()
	crdGK                  = apiextv1.SchemeGroupVersion.WithKind("CustomResourceDefinition").GroupKind()
	validatingWebhookCfgGK = admissionregv1.SchemeGroupVersion.WithKind("ValidatingWebhookConfiguration").GroupKind()
	mutatingWebhookCfgGK   = admissionregv1.SchemeGroupVersion.WithKind("MutatingWebhookConfiguration").GroupKind()
	v1alpha3ScorecardCfgGK = scorecardv1alpha3.GroupVersion.WithKind("Configuration").GroupKind()
)

// UpdateFromDirs adds Roles, ClusterRoles, Deployments, and Custom Resource examples
// found in deployDir, and CustomResourceDefinitions found in crdsDir,
// to their respective fields in a Manifests, then filters and deduplicates them.
// All other objects are added to Manifests.Others.
func (c *Manifests) UpdateFromDirs(deployDir, crdsDir string) error {
	// Collect all manifests in paths.
	err := filepath.Walk(deployDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		scanner := k8sutil.NewYAMLScanner(bytes.NewBuffer(b))
		for scanner.Scan() {
			manifest := scanner.Bytes()
			typeMeta, err := k8sutil.GetTypeMetaFromBytes(manifest)
			if err != nil {
				log.Debugf("No TypeMeta in %s, skipping file", path)
				continue
			}

			gvk := typeMeta.GroupVersionKind()
			switch gvk.GroupKind() {
			case roleGK:
				err = c.addRoles(manifest)
			case clusterRoleGK:
				err = c.addClusterRoles(manifest)
			case roleBindingGK:
				err = c.addRoleBindings(manifest)
			case clusterRoleBindingGK:
				err = c.addClusterRoleBindings(manifest)
			case serviceAccountGK:
				err = c.addServiceAccounts(manifest)
			case serviceGK:
				err = c.addServices(manifest)
			case deploymentGK:
				err = c.addDeployments(manifest)
			case crdGK:
				// Skip for now and add explicitly from CRDsDir input.
			case validatingWebhookCfgGK:
				err = c.addValidatingWebhookConfigurations(manifest)
			case mutatingWebhookCfgGK:
				err = c.addMutatingWebhookConfigurations(manifest)
			case v1alpha3ScorecardCfgGK:
				err = c.addScorecardConfig(manifest)
			default:
				err = c.addOthers(manifest)
			}
			if err != nil {
				return fmt.Errorf("error adding %s to manifest collector: %v", gvk, err)
			}
		}
		return scanner.Err()
	})
	if err != nil {
		return fmt.Errorf("error collecting manifests from directory %s: %v", deployDir, err)
	}

	// Add CRDs from input.
	if isDirExist(crdsDir) {
		c.V1CustomResourceDefinitions, c.V1beta1CustomResourceDefinitions, err = k8sutil.GetCustomResourceDefinitions(crdsDir)
		if err != nil {
			return fmt.Errorf("error adding CustomResourceDefinitions to manifest collector: %v", err)
		}
	}

	// Filter manifests based on data collected.
	c.filter()

	// Remove duplicate manifests.
	if err := c.deduplicate(); err != nil {
		return fmt.Errorf("error removing duplicate manifests: %v", err)
	}

	return nil
}

// UpdateFromReader adds Roles, ClusterRoles, Deployments, CustomResourceDefinitions,
// and Custom Resources found in r to their respective fields in a Manifests, then
// filters and deduplicates them. All other objects are added to Manifests.Others.
func (c *Manifests) UpdateFromReader(r io.Reader) error {
	// Bundle contents.
	scanner := k8sutil.NewYAMLScanner(r)
	for scanner.Scan() {
		manifest := scanner.Bytes()
		typeMeta, err := k8sutil.GetTypeMetaFromBytes(manifest)
		if err != nil {
			log.Debug("No TypeMeta found, skipping manifest")
			continue
		}

		gvk := typeMeta.GroupVersionKind()
		switch gvk.GroupKind() {
		case roleGK:
			err = c.addRoles(manifest)
		case clusterRoleGK:
			err = c.addClusterRoles(manifest)
		case roleBindingGK:
			err = c.addRoleBindings(manifest)
		case clusterRoleBindingGK:
			err = c.addClusterRoleBindings(manifest)
		case serviceAccountGK:
			err = c.addServiceAccounts(manifest)
		case serviceGK:
			err = c.addServices(manifest)
		case deploymentGK:
			err = c.addDeployments(manifest)
		case crdGK:
			err = c.addCustomResourceDefinitions(gvk.Version, manifest)
		case validatingWebhookCfgGK:
			err = c.addValidatingWebhookConfigurations(manifest)
		case mutatingWebhookCfgGK:
			err = c.addMutatingWebhookConfigurations(manifest)
		case v1alpha3ScorecardCfgGK:
			err = c.addScorecardConfig(manifest)
		default:
			err = c.addOthers(manifest)
		}
		if err != nil {
			return fmt.Errorf("error adding %s to manifest collector: %v", gvk, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error collecting manifests from reader: %v", err)
	}

	// Filter manifests based on data collected.
	c.filter()

	// Remove duplicate manifests.
	if err := c.deduplicate(); err != nil {
		return fmt.Errorf("error removing duplicate manifests: %v", err)
	}

	return nil
}

// addRoles assumes all manifest data in rawManifests are Roles and adds them
// to the collector.
func (c *Manifests) addRoles(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		role := rbacv1.Role{}
		if err := yaml.Unmarshal(rawManifest, &role); err != nil {
			return err
		}
		c.Roles = append(c.Roles, role)
	}
	return nil
}

// addClusterRoles assumes all manifest data in rawManifests are ClusterRoles
// and adds them to the collector.
func (c *Manifests) addClusterRoles(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		role := rbacv1.ClusterRole{}
		if err := yaml.Unmarshal(rawManifest, &role); err != nil {
			return err
		}
		c.ClusterRoles = append(c.ClusterRoles, role)
	}
	return nil
}

// addRoleBindings assumes all manifest data in rawManifests are RoleBindings and adds them to the collector.
func (c *Manifests) addRoleBindings(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		binding := rbacv1.RoleBinding{}
		if err := yaml.Unmarshal(rawManifest, &binding); err != nil {
			return err
		}
		c.RoleBindings = append(c.RoleBindings, binding)
	}
	return nil
}

// addClusterRoleBindings assumes all manifest data in rawManifests are ClusterRoleBindings and adds them to the collector.
func (c *Manifests) addClusterRoleBindings(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		binding := rbacv1.ClusterRoleBinding{}
		if err := yaml.Unmarshal(rawManifest, &binding); err != nil {
			return err
		}
		c.ClusterRoleBindings = append(c.ClusterRoleBindings, binding)
	}
	return nil
}

// addServiceAccounts assumes all manifest data in rawManifests are ServiceAccounts and adds them to the collector.
func (c *Manifests) addServiceAccounts(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		sa := corev1.ServiceAccount{}
		if err := yaml.Unmarshal(rawManifest, &sa); err != nil {
			return err
		}
		c.ServiceAccounts = append(c.ServiceAccounts, sa)
	}
	return nil
}

// addServices assumes all manifest data in rawManifests are Services and adds them to the collector.
func (c *Manifests) addServices(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		s := corev1.Service{}
		if err := yaml.Unmarshal(rawManifest, &s); err != nil {
			return err
		}
		c.Services = append(c.Services, s)
	}
	return nil
}

// addDeployments assumes all manifest data in rawManifests are Deployments
// and adds them to the collector.
func (c *Manifests) addDeployments(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		dep := appsv1.Deployment{}
		if err := yaml.Unmarshal(rawManifest, &dep); err != nil {
			return err
		}
		c.Deployments = append(c.Deployments, dep)
	}
	return nil
}

// addCustomResourceDefinitions assumes all manifest data in rawManifests are
// CustomResourceDefinitions and adds them to the collector. version determines
// which CustomResourceDefinition type is used for all manifests in rawManifests.
func (c *Manifests) addCustomResourceDefinitions(version string, rawManifests ...[]byte) (err error) {
	for _, rawManifest := range rawManifests {
		switch version {
		case apiextv1.SchemeGroupVersion.Version:
			crd := apiextv1.CustomResourceDefinition{}
			if err := yaml.Unmarshal(rawManifest, &crd); err != nil {
				return err
			}
			c.V1CustomResourceDefinitions = append(c.V1CustomResourceDefinitions, crd)
		case apiextv1beta1.SchemeGroupVersion.Version:
			crd := apiextv1beta1.CustomResourceDefinition{}
			if err := yaml.Unmarshal(rawManifest, &crd); err != nil {
				return err
			}
			c.V1beta1CustomResourceDefinitions = append(c.V1beta1CustomResourceDefinitions, crd)
		default:
			return fmt.Errorf("unrecognized CustomResourceDefinition version %q", version)
		}
	}
	return nil
}

// addValidatingWebhookConfigurations assumes all manifest data in rawManifests
// are ValidatingWebhookConfigurations and adds their webhooks to the collector.
func (c *Manifests) addValidatingWebhookConfigurations(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		webhookConfig := admissionregv1.ValidatingWebhookConfiguration{}
		if err := yaml.Unmarshal(rawManifest, &webhookConfig); err != nil {
			return err
		}
		c.ValidatingWebhooks = append(c.ValidatingWebhooks, webhookConfig.Webhooks...)
	}
	return nil
}

// addMutatingWebhookConfigurations assumes all manifest data in rawManifests
// are MutatingWebhookConfigurations and adds their webhooks to the collector.
func (c *Manifests) addMutatingWebhookConfigurations(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		webhookConfig := admissionregv1.MutatingWebhookConfiguration{}
		if err := yaml.Unmarshal(rawManifest, &webhookConfig); err != nil {
			return err
		}
		c.MutatingWebhooks = append(c.MutatingWebhooks, webhookConfig.Webhooks...)
	}
	return nil
}

// addScorecardConfig assumes manifest data in rawManifests is a ScorecardConfigs and adds it to the collector.
// If a config has already been found, addScorecardConfig will return an error.
func (c *Manifests) addScorecardConfig(rawManifest []byte) error {
	cfg := scorecardv1alpha3.Configuration{}
	if err := yaml.Unmarshal(rawManifest, &cfg); err != nil {
		return err
	}
	if c.ScorecardConfig.Metadata.Name != "" {
		return errors.New("duplicate scorecard configurations in collector input")
	}
	c.ScorecardConfig = cfg
	return nil
}

// addOthers assumes all manifest data in rawManifests are able to be
// unmarshalled into an Unstructured object and adds them to the collector.
func (c *Manifests) addOthers(rawManifests ...[]byte) error {
	for _, rawManifest := range rawManifests {
		u := unstructured.Unstructured{}
		if err := yaml.Unmarshal(rawManifest, &u); err != nil {
			return err
		}
		c.Others = append(c.Others, u)
	}
	return nil
}

// isDirExist returns true if dir exists on disk.
func isDirExist(dir string) bool {
	if dir == "" {
		return false
	}
	info, err := os.Stat(dir)
	return (err == nil && info.IsDir()) || os.IsExist(err)
}
