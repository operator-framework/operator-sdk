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
	"sync"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold/input"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	pkgk8sutil "github.com/operator-framework/operator-sdk/pkg/k8sutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	crdgen "sigs.k8s.io/controller-tools/pkg/crd"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

// CRD is the input needed to generate a deploy/crds/<group>_<resource>.yaml file
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
	file := fmt.Sprintf("%s_%s.yaml", r.FullGroup, r.Resource)
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
		// controller-tools' generator reads and scaffolds a CRD for the API in
		// {working dir}/pkg/apis/<group>/<version>.
		if err := s.runCRDGenerator(fs); err != nil {
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
		// controller-tools does not set ListKind or Singular names.
		setCRDNamesForResource(crd, s.Resource)
		// As of controller-tools v0.2.0, the CRD generator creates a lengthy
		// "metadata" validation block from ObjectMeta. The simpler "type: object"
		// block is preferable for readability and because ObjectMeta validation
		// is cluster-defined. This block should be overridden as "type: object".
		//
		// Relevant issue:
		// https://github.com/kubernetes-sigs/controller-tools/issues/216
		subCRDValidationMetadata(crd)
	} else {
		// There are currently no commands to update CRD manifests for non-Go
		// operators, so if a CRD manifest already exists for this gvk, this
		// scaffold is a no-op (for now).
		path := crdPathForResource(CRDsDir, s.Resource)
		absPath := filepath.Join(s.AbsProjectPath, path)
		if _, err := s.getFS().Stat(absPath); err == nil {
			b, err := afero.ReadFile(s.getFS(), absPath)
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

	pkgk8sutil.SortVersions(crd.Spec.Versions, pkgk8sutil.GetCRDVersionsName)
	if err := checkCRDVersions(crd); err != nil {
		if _, ok := err.(ErrCRDNoStorageVersion); !ok {
			return nil, err
		}
		setCRDStorageVersion(crd)
	}
	return k8sutil.GetObjectBytes(crd, yaml.Marshal)
}

func (s *CRD) runCRDGenerator(fs ...afero.Fs) (err error) {
	crdFS := afero.NewOsFs()
	if len(fs) == 1 {
		crdFS = fs[0]
	}

	gctx := &genall.GenerationContext{
		Collector: &markers.Collector{
			Registry: &markers.Registry{},
		},
		Checker:    &loader.TypeChecker{},
		InputRule:  genall.InputFromFileSystem,
		OutputRule: crdOutputRule{fs: crdFS},
	}
	absAPIsDir := filepath.Join(s.AbsProjectPath, ApisDir)
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

	gctx.Roots, err = loader.LoadRoots(apiDirs...)
	if err != nil {
		return errors.Wrapf(err, "error loading API roots %+q", apiDirs)
	}

	g := crdgen.Generator{}
	if err := g.RegisterMarkers(gctx.Collector.Registry); err != nil {
		return errors.Wrap(err, "error registering markers for CRD API versions")
	}
	if err := g.Generate(gctx); err != nil {
		return errors.Wrap(err, "error generating a CRD for API versions")
	}
	return nil
}

// subCRDValidationMetadata sets top-level "metadata" validation blocks to
// "type: object".
func subCRDValidationMetadata(crd *apiextv1beta1.CustomResourceDefinition) {
	if crd.Spec.Validation != nil && crd.Spec.Validation.OpenAPIV3Schema != nil {
		if _, ok := crd.Spec.Validation.OpenAPIV3Schema.Properties["metadata"]; ok {
			crd.Spec.Validation.OpenAPIV3Schema.Properties["metadata"] = apiextv1beta1.JSONSchemaProps{
				Type: "object",
			}
		}
	}
	for _, ver := range crd.Spec.Versions {
		if ver.Schema != nil && ver.Schema.OpenAPIV3Schema != nil {
			if _, ok := ver.Schema.OpenAPIV3Schema.Properties["metadata"]; ok {
				ver.Schema.OpenAPIV3Schema.Properties["metadata"] = apiextv1beta1.JSONSchemaProps{
					Type: "object",
				}
			}
		}
	}
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
	if crd.Spec.Version != "" {
		for _, ver := range crd.Spec.Versions {
			if crd.Spec.Version == ver.Name {
				log.Infof("Setting CRD %q storage version to %s", crd.GetName(), ver.Name)
				ver.Storage = true
			}
		}
	} else if len(crd.Spec.Versions) != 0 {
		// Set the first element in spec.versions to storage == true.
		crd.Spec.Versions[0].Storage = true
		log.Infof("Setting CRD %q storage version to %s", crd.GetName(), crd.Spec.Versions[0].Name)
	}
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
		return ErrCRDNoStorageVersion{crd.Spec.Names.Kind}
	}
	return nil
}

type ErrCRDNoStorageVersion struct {
	Kind string
}

func (e ErrCRDNoStorageVersion) Error() string {
	return fmt.Sprintf("spec.versions must have exactly one storage version for CRD %s", e.Kind)
}
