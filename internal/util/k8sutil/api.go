// Copyright 2019 The Operator-SDK Authors
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

package k8sutil

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/operator-framework/operator-registry/pkg/registry"
	log "github.com/sirupsen/logrus"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/yaml"
)

// GetCustomResourceDefinitions returns all CRD manifests of both v1 and v1beta1
// versions in the directory crdsDir. If a duplicate object with different API
// versions is found, and error is returned.
func GetCustomResourceDefinitions(crdsDir string) (
	v1crds []apiextv1.CustomResourceDefinition,
	v1beta1crds []apiextv1beta1.CustomResourceDefinition,
	err error) {

	infos, err := ioutil.ReadDir(crdsDir)
	if err != nil {
		return nil, nil, err
	}

	// The set of all custom resource GVKs in found CRDs.
	crGVKSet := map[schema.GroupVersionKind]struct{}{}
	for _, info := range infos {
		path := filepath.Join(crdsDir, info.Name())

		if info.IsDir() {
			log.Debugf("Skipping dir: %s", path)
			continue
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, nil, fmt.Errorf("error reading manifest %s: %w", path, err)
		}

		scanner := NewYAMLScanner(bytes.NewBuffer(b))
		for scanner.Scan() {
			manifest := scanner.Bytes()
			typeMeta, err := GetTypeMetaFromBytes(manifest)
			if err != nil {
				log.Debugf("Skipping manifest in %s: %v", path, err)
				continue
			}
			if typeMeta.Kind != "CustomResourceDefinition" {
				continue
			}

			// Unmarshal based on CRD version.
			var crGVKs []schema.GroupVersionKind
			switch gvk := typeMeta.GroupVersionKind(); gvk.Version {
			case apiextv1.SchemeGroupVersion.Version:
				crd := apiextv1.CustomResourceDefinition{}
				if err = yaml.Unmarshal(manifest, &crd); err != nil {
					return nil, nil, err
				}
				v1crds = append(v1crds, crd)
				crGVKs = append(crGVKs, GVKsForV1CustomResourceDefinitions(crd)...)
			case apiextv1beta1.SchemeGroupVersion.Version:
				crd := apiextv1beta1.CustomResourceDefinition{}
				if err := yaml.Unmarshal(manifest, &crd); err != nil {
					return nil, nil, err
				}
				v1beta1crds = append(v1beta1crds, crd)
				crGVKs = append(crGVKs, GVKsForV1beta1CustomResourceDefinitions(crd)...)
			default:
				return nil, nil, fmt.Errorf("unrecognized CustomResourceDefinition version %q", gvk.Version)
			}

			// Check if any GVK in crd is a duplicate.
			for _, gvk := range crGVKs {
				if _, hasGVK := crGVKSet[gvk]; hasGVK {
					return nil, nil, fmt.Errorf("duplicate custom resource GVK %s in %s", gvk, path)
				}
				crGVKSet[gvk] = struct{}{}
			}

		}
		if err = scanner.Err(); err != nil {
			return nil, nil, fmt.Errorf("error scanning %s: %w", path, err)
		}
	}
	return v1crds, v1beta1crds, nil
}

// DefinitionsForV1CustomResourceDefinitions returns definition keys for all
// custom resource versions in each crd in crds.
func DefinitionsForV1CustomResourceDefinitions(crds ...apiextv1.CustomResourceDefinition) (keys []registry.DefinitionKey) {
	for _, crd := range crds {
		for _, ver := range crd.Spec.Versions {
			if !ver.Served {
				log.Debugf("Not adding unserved CRD %q version %q to set of owned keys", crd.GetName(), ver.Name)
				continue
			}
			keys = append(keys, registry.DefinitionKey{
				Name:    crd.GetName(),
				Group:   crd.Spec.Group,
				Version: ver.Name,
				Kind:    crd.Spec.Names.Kind,
			})
		}
	}
	return keys
}

// DefinitionsForV1beta1CustomResourceDefinitions returns definition keys for all
// custom resource versions in each crd in crds.
func DefinitionsForV1beta1CustomResourceDefinitions(crds ...apiextv1beta1.CustomResourceDefinition) (keys []registry.DefinitionKey) {
	for _, crd := range crds {
		if len(crd.Spec.Versions) == 0 {
			keys = append(keys, registry.DefinitionKey{
				Name:    crd.GetName(),
				Group:   crd.Spec.Group,
				Version: crd.Spec.Version,
				Kind:    crd.Spec.Names.Kind,
			})
		}
		for _, ver := range crd.Spec.Versions {
			if !ver.Served {
				log.Debugf("Not adding unserved CRD %q version %q to set of owned keys", crd.GetName(), ver.Name)
				continue
			}
			keys = append(keys, registry.DefinitionKey{
				Name:    crd.GetName(),
				Group:   crd.Spec.Group,
				Version: ver.Name,
				Kind:    crd.Spec.Names.Kind,
			})
		}
	}
	return keys
}

// GVKsForV1CustomResourceDefinitions returns GroupVersionKind's for all
// custom resource versions in each crd in crds.
func GVKsForV1CustomResourceDefinitions(crds ...apiextv1.CustomResourceDefinition) (gvks []schema.GroupVersionKind) {
	for _, key := range DefinitionsForV1CustomResourceDefinitions(crds...) {
		gvks = append(gvks, schema.GroupVersionKind{
			Group:   key.Group,
			Version: key.Version,
			Kind:    key.Kind,
		})
	}
	return gvks
}

// GVKsForV1beta1CustomResourceDefinitions returns GroupVersionKind's for all
// custom resource versions in each crd in crds.
func GVKsForV1beta1CustomResourceDefinitions(crds ...apiextv1beta1.CustomResourceDefinition) (gvks []schema.GroupVersionKind) {
	for _, key := range DefinitionsForV1beta1CustomResourceDefinitions(crds...) {
		gvks = append(gvks, schema.GroupVersionKind{
			Group:   key.Group,
			Version: key.Version,
			Kind:    key.Kind,
		})
	}
	return gvks
}

type CRDVersions []apiextv1beta1.CustomResourceDefinitionVersion

func (vs CRDVersions) Len() int { return len(vs) }
func (vs CRDVersions) Less(i, j int) bool {
	return version.CompareKubeAwareVersionStrings(vs[i].Name, vs[j].Name) > 0
}
func (vs CRDVersions) Swap(i, j int) { vs[i], vs[j] = vs[j], vs[i] }

func Convertv1beta1Tov1CustomResourceDefinition(in *apiextv1beta1.CustomResourceDefinition) (*apiextv1.CustomResourceDefinition, error) {
	var unversioned apiext.CustomResourceDefinition
	if err := apiextv1beta1.Convert_v1beta1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(in, &unversioned, nil); err != nil {
		return nil, err
	}

	var out apiextv1.CustomResourceDefinition
	out.TypeMeta.APIVersion = apiextv1.SchemeGroupVersion.String()
	out.TypeMeta.Kind = "CustomResourceDefinition"
	if err := apiextv1.Convert_apiextensions_CustomResourceDefinition_To_v1_CustomResourceDefinition(&unversioned, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}
