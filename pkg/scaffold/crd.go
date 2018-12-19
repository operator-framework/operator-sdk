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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/operator-framework/operator-sdk/pkg/scaffold/input"

	"github.com/ghodss/yaml"
	"github.com/spf13/afero"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crdgenerator "sigs.k8s.io/controller-tools/pkg/crd/generator"
)

// Crd is the input needed to generate a deploy/crds/<group>_<version>_<kind>_crd.yaml file
type Crd struct {
	input.Input

	// Resource defines the inputs for the new custom resource definition
	Resource *Resource
}

func (s *Crd) GetInput() (input.Input, error) {
	if s.Path == "" {
		fileName := fmt.Sprintf("%s_%s_%s_crd.yaml",
			strings.ToLower(s.Resource.Group),
			strings.ToLower(s.Resource.Version),
			s.Resource.LowerKind)
		s.Path = filepath.Join(CrdsDir, fileName)
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
	// Global cache so users can use new Crd structs.
	cache *fsCache
	once  sync.Once
)

func initCache() {
	once.Do(func() {
		cache = &fsCache{Fs: afero.NewMemMapFs()}
	})
}

func (s *Crd) CustomRender() ([]byte, error) {
	i, _ := s.GetInput()
	// controller-tools generates crd file names with no _crd.yaml suffix:
	// <group>_<version>_<kind>.yaml.
	path := strings.Replace(filepath.Base(i.Path), "_crd.yaml", ".yaml", 1)

	// controller-tools' generators read and make crds for all apis in pkg/apis,
	// so generate crds in a cached, in-memory fs to extract the data we need.
	// Note that controller-tools' generator makes different assumptions about
	// how crd field values are structured, so we don't want to use the generated
	// files directly.
	if !cache.fileExists(path) {
		g := &crdgenerator.Generator{
			RootPath:          s.AbsProjectPath,
			Domain:            "placeholder", // Our crds don't use this value.
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

	dstCrd := newCrdForResource(s.Resource)
	var (
		b   []byte
		err error
	)
	// Get our generated crd's from the in-memory fs. If it doesn't exist in the
	// fs, the corresponding API does not exist yet, so scaffold a fresh crd
	// without a validation spec.
	// If it does, and a local crd exists, append the validation spec. Otherwise,
	// generate a fresh crd with the generated validation spec.
	b, err = afero.ReadFile(cache, path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	} else {
		crd := new(apiextv1beta1.CustomResourceDefinition)
		if err = yaml.Unmarshal(b, crd); err != nil {
			return nil, err
		}

		// If the crd exists at i.Path, append the validation spec to its crd spec.
		if _, err := os.Stat(i.Path); err == nil {
			cb, err := ioutil.ReadFile(i.Path)
			if err != nil {
				return nil, err
			}
			if len(cb) > 0 {
				dstCrd = new(apiextv1beta1.CustomResourceDefinition)
				if err = yaml.Unmarshal(cb, dstCrd); err != nil {
					return nil, err
				}
			}
		}
		dstCrd.Spec.Validation = crd.Spec.Validation.DeepCopy()
	}

	return yaml.Marshal(dstCrd)
}

func newCrdForResource(r *Resource) *apiextv1beta1.CustomResourceDefinition {
	return &apiextv1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apiextensions.k8s.io/v1beta1",
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: r.Resource + "." + r.FullGroup,
		},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group: r.FullGroup,
			Names: apiextv1beta1.CustomResourceDefinitionNames{
				Kind:     r.Kind,
				ListKind: r.Kind + "List",
				Plural:   r.Resource,
				Singular: r.LowerKind,
			},
			Scope:   apiextv1beta1.NamespaceScoped,
			Version: r.Version,
		},
	}
}
