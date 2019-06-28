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

package internal

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"

	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olmregistry "github.com/operator-framework/operator-lifecycle-manager/pkg/controller/registry"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

// CatalogSource represents the paths of files containing all data needed to
// construct a CatalogSource object.
type CatalogSource struct {
	ProjectName         string
	Namespace           string
	BundleDir           string
	PackageManifestPath string

	// CatalogSourcePath is an existing CatalogSource manifest to be included
	// in the final combined manifest.
	CatalogSourcePath string

	// Used to decode manifests.
	scheme *runtime.Scheme
}

func wrapBytesErr(err error) error {
	return errors.Wrap(err, "failed to get CatalogSource bytes")
}

// ToConfigMapAndCatalogSource reads all files in s.BundleDir and
// s.PackageManifestPath, combining them into a ConfigMap and CatalogSource.
func (s *CatalogSource) ToConfigMapAndCatalogSource() (*corev1.ConfigMap, *olmapiv1alpha1.CatalogSource, error) {
	if s.BundleDir == "" {
		return nil, nil, fmt.Errorf("bundle dir must be set")
	}
	if s.scheme == nil {
		s.scheme = scheme.Scheme
		if err := addSchemes(s.scheme); err != nil {
			return nil, nil, err
		}
	}

	csvs, crds, pkg, err := s.getBundledObjects()
	if err != nil {
		return nil, nil, wrapBytesErr(err)
	}
	// Users can have all "required" and no "owned" CRD's in their CSV so do not
	// check if crds is empty.
	if len(csvs) == 0 {
		return nil, nil, wrapBytesErr(fmt.Errorf("no CSV's found in bundle dir %s", s.BundleDir))
	}
	if pkg == nil {
		return nil, nil, wrapBytesErr(fmt.Errorf("no package manifest found in bundle dir %s", s.BundleDir))
	}

	csvBytes := []byte{}
	for _, csv := range csvs {
		b, err := yaml.Marshal(csv)
		if err != nil {
			return nil, nil, wrapBytesErr(errors.Wrapf(err, "failed to unmarshal CSV %s", csv.GetName()))
		}
		csvBytes = yamlutil.CombineManifests(csvBytes, b)
	}
	crdBytes := []byte{}
	for _, crd := range crds {
		b, err := yaml.Marshal(crd)
		if err != nil {
			return nil, nil, wrapBytesErr(errors.Wrapf(err, "failed to unmarshal CRD %s", crd.GetName()))
		}
		crdBytes = yamlutil.CombineManifests(crdBytes, b)
	}
	pkgBytes, err := yaml.Marshal(pkg)
	if err != nil {
		return nil, nil, wrapBytesErr(errors.Wrap(err, "failed to unmarshal package manifest"))
	}
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.ToLower(s.ProjectName),
		},
		Data: map[string]string{
			"packages":               string(pkgBytes),
			"clusterServiceVersions": string(csvBytes),
		},
	}
	if s.Namespace != "" {
		configMap.SetNamespace(s.Namespace)
	}
	if len(crdBytes) != 0 {
		configMap.Data["customResourceDefinitions"] = string(crdBytes)
	}
	cs, err := s.getCatalogSource()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get CatalogSource")
	}
	return configMap, cs, nil
}

func addSchemes(s *runtime.Scheme) error {
	if err := apiextv1beta1.AddToScheme(s); err != nil {
		return errors.Wrap(err, "failed to add Kubhernetes API extensions v1beta1 types to scheme")
	}
	if err := olmapiv1alpha1.AddToScheme(s); err != nil {
		return errors.Wrap(err, "failed to add OLM operator API v1alpha2 types to scheme")
	}
	return nil
}

func (s *CatalogSource) getCatalogSource() (cs *olmapiv1alpha1.CatalogSource, err error) {
	name := strings.ToLower(s.ProjectName)
	if s.CatalogSourcePath == "" {
		cs = &olmapiv1alpha1.CatalogSource{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "operators.coreos.com/v1alpha1",
				Kind:       "CatalogSource",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: olmapiv1alpha1.CatalogSourceSpec{
				SourceType:  olmapiv1alpha1.SourceTypeConfigmap,
				ConfigMap:   name,
				DisplayName: k8sutil.GetDisplayName(s.ProjectName),
			},
		}
	} else {
		b, err := ioutil.ReadFile(s.CatalogSourcePath)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read CatalogSource manifest %s", s.CatalogSourcePath)
		}
		dec := serializer.NewCodecFactory(s.scheme).UniversalDeserializer()
		cs, err = decodeCatalogSource(dec, b)
		if err != nil {
			return nil, errors.Wrapf(err, "CatalogSource manifest %s", s.CatalogSourcePath)
		}
	}
	if s.Namespace != "" {
		cs.SetNamespace(s.Namespace)
	}
	return cs, nil
}

func decodeCatalogSource(dec runtime.Decoder, b []byte) (cs *olmapiv1alpha1.CatalogSource, err error) {
	obj, _, err := dec.Decode(b, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode Catalogsource from manifest")
	}
	var ok bool
	if cs, ok = obj.(*olmapiv1alpha1.CatalogSource); !ok {
		return nil, errors.Errorf("object in manifest is not a Catalogsource")
	}
	return cs, nil
}

func decodeCSV(dec runtime.Decoder, b []byte) (csv *olmapiv1alpha1.ClusterServiceVersion, err error) {
	obj, _, err := dec.Decode(b, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode CSV from manifest")
	}
	var ok bool
	if csv, ok = obj.(*olmapiv1alpha1.ClusterServiceVersion); !ok {
		return nil, errors.Errorf("object in manifest is not a CSV")
	}
	return csv, nil
}

func decodeCRD(dec runtime.Decoder, b []byte) (crd *apiextv1beta1.CustomResourceDefinition, err error) {
	obj, _, err := dec.Decode(b, nil, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode CRD from manifest")
	}
	var ok bool
	if crd, ok = obj.(*apiextv1beta1.CustomResourceDefinition); !ok {
		return nil, errors.Errorf("object in manifest is not a CRD")
	}
	return crd, nil
}

// getBundledObjects collects all ClusterServiceVersions,
// CustomResourceDefinitions, and optionally a package manifests in dir.
// If pkgManPath is not empty, that file's data will be used instead of
// any package manifest found in dir.
func (s *CatalogSource) getBundledObjects() (csvs []*olmapiv1alpha1.ClusterServiceVersion, crds []*apiextv1beta1.CustomResourceDefinition, pkg *olmregistry.PackageManifest, err error) {
	infos, err := ioutil.ReadDir(s.BundleDir)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "failed to read bundle dir %s", s.BundleDir)
	}
	if s.PackageManifestPath != "" {
		pkg, err = readPackageManifest(s.PackageManifestPath)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	dec := serializer.NewCodecFactory(s.scheme).UniversalDeserializer()
	for _, info := range infos {
		if !info.IsDir() {
			path := filepath.Join(s.BundleDir, info.Name())
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, nil, nil, errors.Wrapf(err, "failed to read manifest %s", path)
			}
			kind, err := k8sutil.GetKindfromYAML(b)
			if err != nil {
				return nil, nil, nil, errors.Wrapf(err, "failed to get manifest %s Kind", path)
			}
			switch kind {
			case "ClusterServiceVersion":
				csv, err := decodeCSV(dec, b)
				if err != nil {
					return nil, nil, nil, errors.Wrapf(err, "CSV manifest %s", path)
				}
				csvs = append(csvs, csv)
			case "CustomResourceDefinition":
				crd, err := decodeCRD(dec, b)
				if err != nil {
					return nil, nil, nil, errors.Wrapf(err, "CRD manifest %s", path)
				}
				crds = append(crds, crd)
			case "": // Bundled package manifest files do not include a Kind.
				if pkg == nil {
					if kind == "" {
						u := &unstructured.Unstructured{}
						if err := yaml.Unmarshal(b, u); err != nil {
							return nil, nil, nil, errors.Wrapf(err, "failed to unmarshal into unstructured from manifest %s", path)
						}
						// If u does not have a package manifest's required key, skip.
						if _, ok := u.Object["packageName"]; !ok {
							continue
						}
					}
					pkg = &olmregistry.PackageManifest{}
					if err := yaml.Unmarshal(b, pkg); err != nil {
						return nil, nil, nil, errors.Wrapf(err, "failed to unmarshal package manifest from manifest %s", path)
					}
				}
			}
		}
	}

	if err = checkBundleObjects(csvs, crds, pkg); err != nil {
		return nil, nil, nil, errors.Wrapf(err, "bundle dir %s", s.BundleDir)
	}
	return csvs, crds, pkg, nil
}

func readPackageManifest(path string) (*olmregistry.PackageManifest, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read package manifest %s", path)
	}
	pkg := &olmregistry.PackageManifest{}
	if err := yaml.Unmarshal(b, pkg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal package manifest from manifest %s", path)
	}
	return pkg, nil
}

func checkBundleObjects(csvs []*olmapiv1alpha1.ClusterServiceVersion, crds []*apiextv1beta1.CustomResourceDefinition, pkg *olmregistry.PackageManifest) (err error) {
	if len(csvs) == 0 {
		return errors.Errorf("no CSV manifests found in bundle dir")
	}
	csvCRDMap := map[string]map[string]struct{}{}
	for _, csv := range csvs {
		csvCRDMap[csv.GetName()] = map[string]struct{}{}
		for _, o := range csv.Spec.CustomResourceDefinitions.Owned {
			csvCRDMap[csv.GetName()][getCRDDescKey(o)] = struct{}{}
		}
	}
	for _, crd := range crds {
		for _, k := range getCRDKeys(crd) {
			for csvName, owned := range csvCRDMap {
				if _, hasKey := owned[k]; !hasKey {
					return errors.Errorf("bundle dir does not contain owned CRD %s from CSV %s", crd.Spec.Names.Kind, csvName)
				}
			}
		}
	}
	if pkg == nil {
		return errors.Errorf("neither bundle dir nor package manifest path contains a package manifest")
	}
	return nil
}

func getCRDDescKey(crd olmapiv1alpha1.CRDDescription) string {
	return getCRDKey(crd.Version, crd.Kind)
}

// Since version is deprecated, only look at crd.Spec.Versions.
func getCRDKeys(crd *apiextv1beta1.CustomResourceDefinition) (keys []string) {
	for _, v := range crd.Spec.Versions {
		keys = append(keys, getCRDKey(v.Name, crd.Spec.Names.Kind))
	}
	return keys
}

func getCRDKey(v, k string) string {
	return fmt.Sprintf("%s.%s", v, k)
}
