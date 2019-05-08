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

package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"

	"github.com/ghodss/yaml"
	"github.com/spf13/afero"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crdgenerator "sigs.k8s.io/controller-tools/pkg/crd/generator"
)

// CRD is the input needed to generate a deploy/crds/<group>_<version>_<kind>_crd.yaml file
type CRD struct {
	input.Input

	// Resource defines the inputs for the new custom resource definition
	Resource *Resource

	// IsOperatorGo is true when the operator is written in Go.
	IsOperatorGo bool

	once sync.Once
	fs   afero.Fs // For testing, ex. afero.NewMemMapFs()
}

func (s *CRD) initFS(fs afero.Fs) {
	s.once.Do(func() {
		s.fs = fs
	})
}

func (s *CRD) getFS() afero.Fs {
	s.initFS(afero.NewOsFs())
	return s.fs
}

func (s *CRD) GetInput() (input.Input, error) {
	if s.Path == "" {
		fileName := fmt.Sprintf("%s_%s_%s_crd.yaml",
			s.Resource.GoImportGroup,
			strings.ToLower(s.Resource.Version),
			s.Resource.LowerKind)
		s.Path = filepath.Join(CRDsDir, fileName)
	}
	initCache()
	return s.Input, nil
}

type fsCache struct {
	afero.Fs
}

func (c *fsCache) fileExists(path string) bool {
	_, err := c.Stat(path)
	return err == nil
}

var (
	// Global cache so users can use new CRD structs.
	cache *fsCache
	once  sync.Once
)

func initCache() {
	once.Do(func() {
		cache = &fsCache{Fs: afero.NewMemMapFs()}
	})
}

var _ CustomRenderer = &CRD{}

func (s *CRD) SetFS(fs afero.Fs) { s.initFS(fs) }

func (s *CRD) CustomRender() ([]byte, error) {
	i, err := s.GetInput()
	if err != nil {
		return nil, err
	}

	crd := &apiextv1beta1.CustomResourceDefinition{}
	if s.IsOperatorGo {
		// controller-tools generates crd file names with no _crd.yaml suffix:
		// <group>_<version>_<kind>.yaml.
		path := strings.Replace(filepath.Base(i.Path), "_crd.yaml", ".yaml", 1)

		// controller-tools' generators read and make crds for all apis in pkg/apis,
		// so generate crds in a cached, in-memory fs to extract the data we need.
		if !cache.fileExists(path) {
			g := &crdgenerator.Generator{
				RootPath:          s.AbsProjectPath,
				Domain:            strings.SplitN(s.Resource.FullGroup, ".", 2)[1],
				Repo:              s.Repo,
				OutputDir:         ".",
				SkipMapValidation: false,
				OutFs:             cache,
			}
			if err := g.ValidateAndInitFields(); err != nil {
				return nil, err
			}
			if err := g.Do(); err != nil {
				return nil, err
			}
		}

		b, err := afero.ReadFile(cache, path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("no API exists for Group %s Version %s Kind %s",
					s.Resource.GoImportGroup, s.Resource.Version, s.Resource.Kind)
			}
			return nil, err
		}
		if err = yaml.Unmarshal(b, crd); err != nil {
			return nil, err
		}
		// controller-tools does not set ListKind or Singular names.
		setCRDNamesForResource(crd, s.Resource)
		// Remove controller-tools default label.
		delete(crd.Labels, "controller-tools.k8s.io")
	} else {
		// There are currently no commands to update CRD manifests for non-Go
		// operators, so if a CRD manifests already exists for this gvk, this
		// scaffold is a no-op.
		path := filepath.Join(s.AbsProjectPath, i.Path)
		if _, err = s.getFS().Stat(path); err == nil {
			b, err := afero.ReadFile(s.getFS(), path)
			if err != nil {
				return nil, err
			}
			if len(b) == 0 {
				crd = newCRDForResource(s.Resource)
			} else {
				if err = yaml.Unmarshal(b, crd); err != nil {
					return nil, err
				}
			}
		}
	}

	setCRDVersions(crd)
	return k8sutil.GetObjectBytes(crd)
}

func newCRDForResource(r *Resource) *apiextv1beta1.CustomResourceDefinition {
	crd := &apiextv1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1beta1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Resource + "." + r.FullGroup,
		},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group:   r.FullGroup,
			Scope:   apiextv1beta1.NamespaceScoped,
			Version: r.Version,
			Subresources: &apiextv1beta1.CustomResourceSubresources{
				Status: &apiextv1beta1.CustomResourceSubresourceStatus{},
			},
		},
	}
	setCRDNamesForResource(crd, r)
	return crd
}

func setCRDNamesForResource(crd *apiextv1beta1.CustomResourceDefinition, r *Resource) {
	if crd.Spec.Names.Kind == "" {
		crd.Spec.Names.Kind = r.Kind
	}
	if crd.Spec.Names.ListKind == "" {
		crd.Spec.Names.ListKind = r.Kind + "List"
	}
	if crd.Spec.Names.Plural == "" {
		crd.Spec.Names.Plural = r.Resource
	}
	if crd.Spec.Names.Singular == "" {
		crd.Spec.Names.Singular = r.LowerKind
	}
}

func setCRDVersions(crd *apiextv1beta1.CustomResourceDefinition) {
	// crd.Version is deprecated, use crd.Versions instead.
	var crdVersions []apiextv1beta1.CustomResourceDefinitionVersion
	if crd.Spec.Version != "" {
		var verExists, hasStorageVer bool
		for _, ver := range crd.Spec.Versions {
			if crd.Spec.Version == ver.Name {
				verExists = true
			}
			// There must be exactly one version flagged as a storage version.
			if ver.Storage {
				hasStorageVer = true
			}
		}
		if !verExists {
			crdVersions = []apiextv1beta1.CustomResourceDefinitionVersion{
				{Name: crd.Spec.Version, Served: true, Storage: !hasStorageVer},
			}
		}
	} else {
		crdVersions = []apiextv1beta1.CustomResourceDefinitionVersion{
			{Name: "v1alpha1", Served: true, Storage: true},
		}
	}

	if len(crd.Spec.Versions) > 0 {
		// crd.Version should always be the first element in crd.Versions.
		crd.Spec.Versions = append(crdVersions, crd.Spec.Versions...)
	} else {
		crd.Spec.Versions = crdVersions
	}
}
