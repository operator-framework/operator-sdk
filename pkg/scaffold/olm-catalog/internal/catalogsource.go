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

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
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
)

type CatalogSource struct {
	ProjectName         string
	BundleDir           string
	PackageManifestPath string
}

func wrapBytesErr(err error) error {
	return errors.Wrap(err, "get CatalogSource bytes")
}

func (s *CatalogSource) Bytes() ([]byte, error) {
	lowerProjName := strings.ToLower(s.ProjectName)
	if s.BundleDir == "" {
		s.BundleDir = filepath.Join(scaffold.OLMCatalogDir, lowerProjName)
	}

	csv, crds, pkg, err := readBundleDir(s.BundleDir, s.PackageManifestPath)
	if err != nil {
		return nil, wrapBytesErr(err)
	}
	// Users can have all "required" and no "owned" CRD's in their CSV so do not
	// check if crds is empty.
	if csv == nil {
		return nil, wrapBytesErr(fmt.Errorf("no CSV found in bundle dir %s", s.BundleDir))
	}
	if pkg == nil {
		return nil, wrapBytesErr(fmt.Errorf("no package manifest found in bundle dir %s", s.BundleDir))
	}

	csvBytes, err := yaml.Marshal(csv)
	if err != nil {
		return nil, wrapBytesErr(errors.Wrap(err, "unmarshal CSV"))
	}
	crdBytes := []byte{}
	for _, crd := range crds {
		b, err := yaml.Marshal(crd)
		if err != nil {
			return nil, wrapBytesErr(errors.Wrap(err, "unmarshal CRD"))
		}
		crdBytes = yamlutil.CombineManifests(crdBytes, b)
	}
	pkgBytes, err := yaml.Marshal(pkg)
	if err != nil {
		return nil, wrapBytesErr(errors.Wrap(err, "unmarshal package manifest"))
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
	if len(crdBytes) != 0 {
		configMap.Data["customResourceDefinitions"] = string(crdBytes)
	}
	return k8sutil.GetObjectBytes(configMap)
}

func readBundleDir(dir string, pkgManPath ...string) (csv *olmapiv1alpha1.ClusterServiceVersion, crds []*apiextv1beta1.CustomResourceDefinition, pkg *olmregistry.PackageManifest, err error) {
	infos, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, nil, nil, errors.Wrapf(err, "read bundle dir %s", dir)
	}
	if len(pkgManPath) != 0 && pkgManPath[0] != "" {
		p := pkgManPath[0]
		b, err := ioutil.ReadFile(p)
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "read package manifest %s", p)
		}
		pkg = &olmregistry.PackageManifest{}
		if err := yaml.Unmarshal(b, pkg); err != nil {
			return nil, nil, nil, errors.Wrapf(err, "unmarshal package manifest from manifest %s", p)
		}
	}
	for _, info := range infos {
		if !info.IsDir() {
			b, err := ioutil.ReadFile(filepath.Join(dir, info.Name()))
			if err != nil {
				return nil, nil, nil, errors.Wrapf(err, "read manifest %s", info.Name())
			}
			kind, err := k8sutil.GetKindfromYAML(b)
			if err != nil {
				return nil, nil, nil, errors.Wrapf(err, "get manifest %s Kind", info.Name())
			}
			switch kind {
			case "ClusterServiceVersion":
				csv = &olmapiv1alpha1.ClusterServiceVersion{}
				if err := yaml.Unmarshal(b, csv); err != nil {
					return nil, nil, nil, errors.Wrapf(err, "unmarshal CSV from manifest %s", info.Name())
				}
			case "CustomResourceDefinition":
				crd := &apiextv1beta1.CustomResourceDefinition{}
				if err := yaml.Unmarshal(b, crd); err != nil {
					return nil, nil, nil, errors.Wrapf(err, "unmarshal CRD from manifest %s", info.Name())
				}
				crds = append(crds, crd)
			case "", "PackageManifest":
				if pkg == nil {
					// Many package manifest files do not include a Kind.
					if kind == "" {
						u := &unstructured.Unstructured{}
						if err := yaml.Unmarshal(b, u); err != nil {
							return nil, nil, nil, errors.Wrapf(err, "unmarshal into map from manifest %s", info.Name())
						}
						// If u does not have a package manifest's required key, skip.
						if _, ok := u.Object["packageName"]; !ok {
							continue
						}
					}
					pkg = &olmregistry.PackageManifest{}
					if err := yaml.Unmarshal(b, pkg); err != nil {
						return nil, nil, nil, errors.Wrapf(err, "unmarshal package manifest from manifest %s", info.Name())
					}
				}
			default:
				continue
			}
		}
	}
	return csv, crds, pkg, nil
}
