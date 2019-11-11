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
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crdgen "sigs.k8s.io/controller-tools/pkg/crd"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/loader"
)

// CRD is the input needed to generate a deploy/crds/<full group>_<resource>_crd.yaml file
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
		s.Path = crdPathForResource(CRDsDir, s.Resource)
	}
	return s.Input, nil
}

func crdPathForResource(dir string, r *Resource) string {
	file := fmt.Sprintf("%s_%s_crd.yaml", r.FullGroup, r.Resource)
	return filepath.Join(dir, file)
}

type crdOutputRule struct {
	fs afero.Fs
}

var _ genall.OutputRule = crdOutputRule{}

// Open is meant to be used to generate a CRD manifest in memory at path.
func (o crdOutputRule) Open(_ *loader.Package, path string) (io.WriteCloser, error) {
	if o.fs == nil {
		return nil, errors.Errorf("error opening %s: crdOutputRule fs must be set", path)
	}
	return o.fs.Create(path)
}

var _ CustomRenderer = &CRD{}

func (s *CRD) SetFS(fs afero.Fs) { s.initFS(fs) }

func (s *CRD) CustomRender() ([]byte, error) {
	crd := &apiextv1beta1.CustomResourceDefinition{}
	if s.IsOperatorGo {
		fs := afero.NewMemMapFs()
		// controller-tool's generator reads and scaffolds a CRD for all APIs in
		// pkg/apis.
		err := runCRDGenerator(crdOutputRule{fs: fs}, s.AbsProjectPath)
		if err != nil {
			return nil, err
		}
		// controller-tools generates CRD file names in the format below, which
		// we need to read from fs.
		genFile := fmt.Sprintf("%s_%s.yaml", s.Resource.FullGroup, s.Resource.Resource)
		b, err := afero.ReadFile(fs, genFile)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("error generating CRD for Group %s Version %s Kind %s",
					s.Resource.GoImportGroup, s.Resource.Version, s.Resource.Kind)
			}
			return nil, err
		}
		if err = yaml.Unmarshal(b, crd); err != nil {
			return nil, err
		}

		// controller-tools inserts an annotation and assumes that the binary
		// that creates the CRD is controller-gen. In this case, we don't use
		// controller-gen. Instead, we vendor and use the same library that
		// controller-gen does.
		//
		// The value that gets populated in the annotation is based on the
		// build info of the compiled binary, not on the version of the
		// vendored controller-tools library.
		//
		// See: https://github.com/kubernetes-sigs/controller-tools/issues/348
		//
		// TODO(joelanford): Sort out what to do with this. Until then, let's
		// just remove it.
		delete(crd.Annotations, "controller-gen.kubebuilder.io/version")
	} else {
		// There are currently no commands to update CRD manifests for non-Go
		// operators, so if a CRD manifest already exists for this gvk, this
		// scaffold is a no-op (for now).
		path := crdPathForResource(filepath.Join(s.AbsProjectPath, CRDsDir), s.Resource)
		if _, err := s.getFS().Stat(path); err == nil {
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

	setCRDStorageVersion(crd)
	if err := checkCRDVersions(crd); err != nil {
		return nil, err
	}
	sort.Sort(k8sutil.CRDVersions(crd.Spec.Versions))
	return k8sutil.GetObjectBytes(crd, yaml.Marshal)
}

func runCRDGenerator(rule genall.OutputRule, root string) (err error) {
	absAPIsDir := filepath.Join(root, ApisDir)
	gvs, err := k8sutil.ParseGroupVersions(absAPIsDir)
	if err != nil {
		return errors.Wrapf(err, "error parsing API group versions from directory %+q", absAPIsDir)
	}
	apiDirs := []string{}
	for g, vs := range gvs {
		for _, v := range vs {
			apiDirs = append(apiDirs, filepath.Join(absAPIsDir, g, v))
		}
	}

	cg := crdgen.Generator{}
	gens := genall.Generators{cg}
	r, err := gens.ForRoots(apiDirs...)
	if err != nil {
		return errors.Wrapf(err, "error loading API roots %+q", apiDirs)
	}
	r.OutputRules.ByGenerator = map[genall.Generator]genall.OutputRule{cg: rule}
	ctx := r.GenerationContext
	ctx.OutputRule = r.OutputRules.ForGenerator(gens[0])
	if err := gens[0].Generate(&ctx); err != nil {
		return errors.Wrapf(err, "error generating CRDs")
	}
	return nil
}

func newCRDForResource(r *Resource) *apiextv1beta1.CustomResourceDefinition {
	crd := &apiextv1beta1.CustomResourceDefinition{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiextv1beta1.SchemeGroupVersion.String(),
			Kind:       "CustomResourceDefinition",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s.%s", r.Resource, r.FullGroup),
		},
		Spec: apiextv1beta1.CustomResourceDefinitionSpec{
			Group: r.FullGroup,
			Scope: apiextv1beta1.NamespaceScoped,
			Versions: []apiextv1beta1.CustomResourceDefinitionVersion{
				{Name: r.Version, Served: true, Storage: true},
			},
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

func setCRDStorageVersion(crd *apiextv1beta1.CustomResourceDefinition) {
	if len(crd.Spec.Versions) == 0 {
		return
	}
	for _, ver := range crd.Spec.Versions {
		if ver.Storage {
			return
		}
	}
	// Set the first element in spec.versions to be the storage version.
	log.Infof("Setting CRD %q storage version to %s", crd.GetName(), crd.Spec.Versions[0].Name)
	crd.Spec.Versions[0].Storage = true
}

// checkCRDVersions ensures version(s) generated for a CRD are in valid format.
// From the Kubernetes CRD docs:
//
// The version field is deprecated and optional, but if it is not empty,
// it must match the first item in the versions field.
func checkCRDVersions(crd *apiextv1beta1.CustomResourceDefinition) error {
	singleVer := crd.Spec.Version != ""
	multiVers := len(crd.Spec.Versions) > 0
	if singleVer {
		if !multiVers {
			log.Warnf("CRD %s: spec.version is deprecated and should be migrated to spec.versions", crd.Spec.Names.Kind)
		} else if crd.Spec.Version != crd.Spec.Versions[0].Name {
			return errors.Errorf("spec.version %s must be the first element in spec.versions for CRD %s", crd.Spec.Version, crd.Spec.Names.Kind)
		}
	}

	var hasStorageVer bool
	for _, ver := range crd.Spec.Versions {
		// There must be exactly one version flagged as a storage version.
		if ver.Storage {
			if hasStorageVer {
				return errors.Errorf("spec.versions cannot have more than one storage version for CRD %s", crd.Spec.Names.Kind)
			}
			hasStorageVer = true
		}
	}
	if multiVers && !hasStorageVer {
		return errors.Errorf("spec.versions must have exactly one storage version for CRD %s", crd.Spec.Names.Kind)
	}
	return nil
}
