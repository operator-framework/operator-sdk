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

package catalog

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	registryutil "github.com/operator-framework/operator-sdk/internal/util/registry"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"

	"github.com/ghodss/yaml"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

// CatalogSourceBundle represents the paths of files containing all data
// needed to construct a combined CatalogSource and ConfigMap object.
type CatalogSourceBundle struct {
	ProjectName         string
	Namespace           string
	BundleDir           string
	PackageManifestPath string
	// CatalogSourcePath is an existing CatalogSource manifest to be included
	// in the final combined manifest.
	CatalogSourcePath string
}

func wrapBytesErr(err error) error {
	return errors.Wrap(err, "failed to get CatalogSourceBundle bytes")
}

// ToConfigMapAndCatalogSource reads all files in s.BundleDir and
// s.PackageManifestPath, combining them into a ConfigMap and CatalogSource.
func (s *CatalogSourceBundle) ToConfigMapAndCatalogSource() (*corev1.ConfigMap, *olmapiv1alpha1.CatalogSource, error) {
	bundle := registryutil.Bundle{
		BundleDir:           s.BundleDir,
		PackageManifestPath: s.PackageManifestPath,
	}
	csvs, crds, pkg, err := bundle.GetBundledObjects()
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
			APIVersion: corev1.SchemeGroupVersion.String(),
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

func (s *CatalogSourceBundle) getCatalogSource() (cs *olmapiv1alpha1.CatalogSource, err error) {
	name := strings.ToLower(s.ProjectName)
	if s.CatalogSourcePath == "" {
		cs = &olmapiv1alpha1.CatalogSource{
			TypeMeta: metav1.TypeMeta{
				APIVersion: olmapiv1alpha1.SchemeGroupVersion.String(),
				Kind:       olmapiv1alpha1.CatalogSourceKind,
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
		sch := scheme.Scheme
		if err = olmapiv1alpha1.AddToScheme(sch); err != nil {
			return nil, errors.Wrap(err, "failed to add OLM operator API v1alpha1 types to scheme")
		}
		dec := serializer.NewCodecFactory(sch).UniversalDeserializer()
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
		return nil, errors.Wrapf(err, "failed to decode CatalogSource from manifest")
	}
	var ok bool
	if cs, ok = obj.(*olmapiv1alpha1.CatalogSource); !ok {
		return nil, errors.Errorf("object in manifest is not a Catalogsource")
	}
	return cs, nil
}
