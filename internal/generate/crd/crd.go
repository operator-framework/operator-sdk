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

package crd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	genutil "github.com/operator-framework/operator-sdk/internal/generate/util"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// crdGenerator configures the CustomResourceDefintion manifest generator
// for Go and non-Go projects.
type crdGenerator struct {
	genutil.Config
	// isOperatorGo is true when the operator is written in Go.
	isOperatorGo bool
	// resource contains API information used to configure single-CRD generation.
	// This is only required when isOperatorGo is false.
	resource scaffold.Resource
}

// NewCRDGo returns a CRD generator configured to generate CustomResourceDefintion
// manifests from Go API files.
func NewCRDGo(cfg genutil.Config) genutil.Generator {
	g := crdGenerator{
		Config:       cfg,
		isOperatorGo: true,
	}
	if g.InputDir == "" {
		g.InputDir = scaffold.ApisDir
	}
	if g.OutputDir == "" {
		g.OutputDir = scaffold.CRDsDir
	}
	return g
}

// NewCRDGo returns a CRD generator configured to generate a
// CustomResourceDefintion manifest from scratch using data in resource.
func NewCRDNonGo(cfg genutil.Config, resource scaffold.Resource) genutil.Generator {
	g := crdGenerator{
		Config:       cfg,
		resource:     resource,
		isOperatorGo: false,
	}
	if g.InputDir == "" {
		g.InputDir = scaffold.CRDsDir
	}
	if g.OutputDir == "" {
		g.OutputDir = scaffold.CRDsDir
	}
	return g
}

func (g crdGenerator) validate() error {
	if g.InputDir == "" {
		return errors.New("input dir cannot be empty")
	}
	if g.InputDir == "" {
		return errors.New("output dir cannot be empty")
	}
	if !g.isOperatorGo {
		if err := g.resource.Validate(); err != nil {
			return errors.Wrap(err, "resource is invalid:")
		}
	}
	return nil
}

// Generate generates CRD manifests and writes them to g.OutputDir.
func (g crdGenerator) Generate() (err error) {
	if err = g.validate(); err != nil {
		return errors.Wrap(err, "validation error")
	}
	var fileMap map[string][]byte
	if g.isOperatorGo {
		fileMap, err = g.generateGo()
	} else {
		fileMap, err = g.generateNonGo()
	}
	if err != nil {
		return errors.Wrap(err, "error generating CRD manifests")
	}
	if err = os.MkdirAll(g.OutputDir, fileutil.DefaultDirFileMode); err != nil {
		return errors.Wrapf(err, "error mkdir %s", g.OutputDir)
	}
	for fileName, b := range fileMap {
		path := filepath.Join(g.OutputDir, fileName)
		if err := ioutil.WriteFile(path, b, fileutil.DefaultFileMode); err != nil {
			return errors.Wrap(err, "error writing CRD manifests")
		}
	}
	return nil
}

func getFileNameForResource(r scaffold.Resource) string {
	return fmt.Sprintf("%s_%s_crd.yaml", r.FullGroup, r.Resource)
}

// generateNonGo generates CRDs for Go projects using Go API files.
func (g crdGenerator) generateGo() (map[string][]byte, error) {
	fileMap := map[string][]byte{}
	// Generate files in the generator's cache so we can modify the file name
	// and annotations.
	defName := "output:crd:cache"
	cacheOutputDir := string(filepath.Separator) + filepath.Clean(g.OutputDir)
	rawOpts := []string{
		"crd",
		fmt.Sprintf("paths=%s/...", fileutil.DotPath(g.InputDir)),
		fmt.Sprintf("%s:dir=%s", defName, cacheOutputDir),
	}
	cachedGen := genutil.NewCachedGenerator()
	cachedGen.AddOutputRule(defName, genutil.OutputToCachedDirectory{})
	if err := cachedGen.Run(rawOpts); err != nil {
		return nil, err
	}
	cache := genutil.GetCache()
	infos, err := afero.ReadDir(cache, cacheOutputDir)
	if err != nil {
		return nil, err
	}
	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		path := filepath.Join(cacheOutputDir, info.Name())
		b, err := afero.ReadFile(cache, path)
		if err != nil {
			return nil, err
		}
		scanner := yamlutil.NewYAMLScanner(b)
		modifiedCRD := []byte{}
		for scanner.Scan() {
			sb := scanner.Bytes()
			typeMeta, err := k8sutil.GetTypeMetaFromBytes(sb)
			if err != nil {
				return nil, err
			}
			if typeMeta.Kind != "CustomResourceDefinition" {
				continue
			}
			crd := &apiextv1beta1.CustomResourceDefinition{}
			if err = yaml.Unmarshal(sb, crd); err != nil {
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
			if sb, err = k8sutil.GetObjectBytes(crd, yaml.Marshal); err != nil {
				return nil, err
			}
			modifiedCRD = yamlutil.CombineManifests(modifiedCRD, sb)
		}
		if err = scanner.Err(); err != nil {
			return nil, err
		}
		if len(modifiedCRD) != 0 {
			// Until we bump dependencies to Kubernetes v1.16, generated validation
			// descriptions for kind and apiVersion will contain an invalid link.
			// Manually replace them here.
			//
			// TODO(estroz): remove on k8s v1.16 bump.
			modifiedCRD = bytes.ReplaceAll(modifiedCRD,
				[]byte("https://git.k8s.io/community/contributors/devel/api-conventions.md"),
				[]byte("https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md"),
			)
			fileNameNoExt := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
			fileMap[fileNameNoExt+"_crd.yaml"] = modifiedCRD
		}
	}
	if len(fileMap) == 0 {
		return nil, errors.New("no generated files found")
	}
	return fileMap, nil
}

// generateNonGo generates a CRD for non-Go projects using a resource.
func (g crdGenerator) generateNonGo() (map[string][]byte, error) {
	crd := &apiextv1beta1.CustomResourceDefinition{}
	fileMap := map[string][]byte{}
	fileName := getFileNameForResource(g.resource)
	path := filepath.Join(g.InputDir, fileName)
	if _, err := os.Stat(path); err == nil {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err = yaml.Unmarshal(b, crd); err != nil {
			return nil, err
		}
		// If version is new, append it to spec.versions.
		hasVersion := false
		for _, version := range crd.Spec.Versions {
			if version.Name == g.resource.Version {
				hasVersion = true
				break
			}
		}
		if !hasVersion {
			// Let either the user or below logic determine whether this new
			// version is stored or not.
			crd.Spec.Versions = append(crd.Spec.Versions, apiextv1beta1.CustomResourceDefinitionVersion{
				Name:    g.resource.Version,
				Storage: false,
				Served:  true,
			})
		}
	} else if os.IsNotExist(err) {
		crd = newCRDForResource(g.resource)
	} else {
		return nil, err
	}

	sort.Sort(k8sutil.CRDVersions(crd.Spec.Versions))
	setCRDStorageVersion(crd)
	if err := checkCRDVersions(crd); err != nil {
		return nil, err
	}
	b, err := k8sutil.GetObjectBytes(crd, yaml.Marshal)
	if err != nil {
		return nil, err
	}
	fileMap[fileName] = b
	return fileMap, nil
}

// newCRDForResource constructs a barebones CRD using data in resource.
func newCRDForResource(r scaffold.Resource) *apiextv1beta1.CustomResourceDefinition {
	return &apiextv1beta1.CustomResourceDefinition{
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
			Names: apiextv1beta1.CustomResourceDefinitionNames{
				Kind:     r.Kind,
				ListKind: r.Kind + "List",
				Plural:   r.Resource,
				Singular: r.LowerKind,
			},
			Subresources: &apiextv1beta1.CustomResourceSubresources{
				Status: &apiextv1beta1.CustomResourceSubresourceStatus{},
			},
		},
	}
}

// setCRDStorageVersion sets exactly one version's storage field to true if
// one is not already set.
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
