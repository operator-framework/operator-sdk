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
	"crypto/sha256"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// filter applies filtering rules to certain manifest types in a collection.
func (c *Manifests) filter() {
	c.filterCustomResources()
}

// filterCustomResources filters "other" objects, which contain likely
// Custom Resources corresponding to a CustomResourceDefinition, by GVK.
func (c *Manifests) filterCustomResources() {
	crdGVKSet := make(map[schema.GroupVersionKind]struct{})
	v1crdGVKs := k8sutil.GVKsForV1CustomResourceDefinitions(c.V1CustomResourceDefinitions...)
	v1beta1crdGVKs := k8sutil.GVKsForV1beta1CustomResourceDefinitions(c.V1beta1CustomResourceDefinitions...)
	for _, gvk := range append(v1crdGVKs, v1beta1crdGVKs...) {
		crdGVKSet[gvk] = struct{}{}
	}

	customResources := []unstructured.Unstructured{}
	for _, other := range c.Others {
		if _, gvkMatches := crdGVKSet[other.GroupVersionKind()]; gvkMatches {
			customResources = append(customResources, other)
		}
	}
	c.CustomResources = customResources
}

// deduplicate removes duplicate objects from the collection, since we are
// collecting an arbitrary list of manifests.
func (c *Manifests) deduplicate() error {
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

	v1crds := []apiextv1.CustomResourceDefinition{}
	for _, crd := range c.V1CustomResourceDefinitions {
		hasHash, err := addToHashes(&crd, hashes)
		if err != nil {
			return err
		}
		if !hasHash {
			v1crds = append(v1crds, crd)
		}
	}
	c.V1CustomResourceDefinitions = v1crds

	v1beta1crds := []apiextv1beta1.CustomResourceDefinition{}
	for _, crd := range c.V1beta1CustomResourceDefinitions {
		hasHash, err := addToHashes(&crd, hashes)
		if err != nil {
			return err
		}
		if !hasHash {
			v1beta1crds = append(v1beta1crds, crd)
		}
	}
	c.V1beta1CustomResourceDefinitions = v1beta1crds

	validatingWebhooks := []admissionregv1.ValidatingWebhook{}
	for _, webhook := range c.ValidatingWebhooks {
		hasHash, err := addToHashes(&webhook, hashes)
		if err != nil {
			return err
		}
		if !hasHash {
			validatingWebhooks = append(validatingWebhooks, webhook)
		}
	}
	c.ValidatingWebhooks = validatingWebhooks

	mutatingWebhooks := []admissionregv1.MutatingWebhook{}
	for _, webhook := range c.MutatingWebhooks {
		hasHash, err := addToHashes(&webhook, hashes)
		if err != nil {
			return err
		}
		if !hasHash {
			mutatingWebhooks = append(mutatingWebhooks, webhook)
		}
	}
	c.MutatingWebhooks = mutatingWebhooks

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
