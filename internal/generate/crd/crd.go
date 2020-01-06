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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gen "github.com/operator-framework/operator-sdk/internal/generate/gen"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
	"github.com/operator-framework/operator-sdk/internal/util/yamlutil"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// crdGenerator configures the CustomResourceDefintion manifest generator
// for Go and non-Go projects.
type crdGenerator struct {
	gen.Config
	// isOperatorGo is true when the operator is written in Go.
	isOperatorGo bool
	// resource contains API information used to configure single-CRD generation.
	// This is only required when isOperatorGo is false.
	resource scaffold.Resource
}

const (
	APIsDirKey = "apis"
	CRDsDirKey = "crds"
)

// NewCRDGo returns a CRD generator configured to generate CustomResourceDefintion
// manifests from Go API files.
func NewCRDGo(cfg gen.Config) gen.Generator {
	g := crdGenerator{
		Config:       cfg,
		isOperatorGo: true,
	}
	if g.Inputs == nil {
		g.Inputs = map[string]string{}
	}
	if crdsDir, ok := g.Inputs[CRDsDirKey]; !ok || crdsDir == "" {
		g.Inputs[CRDsDirKey] = scaffold.CRDsDir
	}
	if apisDir, ok := g.Inputs[APIsDirKey]; !ok || apisDir == "" {
		g.Inputs[APIsDirKey] = scaffold.ApisDir
	}
	if g.OutputDir == "" {
		g.OutputDir = g.Inputs[CRDsDirKey]
	}
	return g
}

// NewCRDNonGo returns a CRD generator configured to generate a
// CustomResourceDefintion manifest from scratch using data in resource.
func NewCRDNonGo(cfg gen.Config, resource scaffold.Resource) gen.Generator {
	g := crdGenerator{
		Config:       cfg,
		resource:     resource,
		isOperatorGo: false,
	}
	if g.Inputs == nil {
		g.Inputs = map[string]string{}
	}
	if crdsDir, ok := g.Inputs[CRDsDirKey]; !ok || crdsDir == "" {
		g.Inputs[CRDsDirKey] = scaffold.CRDsDir
	}
	if g.OutputDir == "" {
		g.OutputDir = g.Inputs[CRDsDirKey]
	}
	return g
}

func (g crdGenerator) validate() error {
	if len(g.Inputs) == 0 {
		return errors.New("inputs cannot be empty")
	}
	if _, ok := g.Inputs[CRDsDirKey]; !ok {
		return errors.New("input CRDs dir cannot be empty")
	}
	if g.isOperatorGo {
		if _, ok := g.Inputs[APIsDirKey]; !ok {
			return errors.New("input APIs dir cannot be empty")
		}
	}
	if g.OutputDir == "" {
		return errors.New("output dir cannot be empty")
	}
	if !g.isOperatorGo {
		if err := g.resource.Validate(); err != nil {
			return fmt.Errorf("resource is invalid: %w", err)
		}
	}
	return nil
}

// Generate generates CRD manifests and writes them to g.OutputDir.
func (g crdGenerator) Generate() (err error) {
	if err = g.validate(); err != nil {
		return fmt.Errorf("error validating generator configuration: %w", err)
	}
	var fileMap map[string][]byte
	if g.isOperatorGo {
		fileMap, err = g.generateGo()
	} else {
		fileMap, err = g.generateNonGo()
	}
	if err != nil {
		return fmt.Errorf("error generating CRD manifests: %w", err)
	}
	if err = os.MkdirAll(g.OutputDir, fileutil.DefaultDirFileMode); err != nil {
		return fmt.Errorf("error mkdir %s: %w", g.OutputDir, err)
	}
	for fileName, b := range fileMap {
		path := filepath.Join(g.OutputDir, fileName)
		if err := ioutil.WriteFile(path, b, fileutil.DefaultFileMode); err != nil {
			return fmt.Errorf("error writing CRD manifests: %w", err)
		}
	}
	return nil
}

func getFileNameForResource(r scaffold.Resource) string {
	return fmt.Sprintf("%s_%s_crd.yaml", r.FullGroup, r.Resource)
}

// generateGo generates CRDs for Go projects using Go API files.
func (g crdGenerator) generateGo() (map[string][]byte, error) {
	fileMap := map[string][]byte{}
	// Generate files in the generator's cache so we can modify the file name
	// and annotations.
	defName := "output:crd:cache"
	cacheOutputDir := string(filepath.Separator) + filepath.Clean(g.OutputDir)
	rawOpts := []string{
		"crd",
		fmt.Sprintf("paths=%s/...", fileutil.DotPath(g.Inputs[APIsDirKey])),
		fmt.Sprintf("%s:dir=%s", defName, cacheOutputDir),
	}
	runner := gen.NewCachedRunner()
	runner.AddOutputRule(defName, gen.OutputToCachedDirectory{})
	if err := runner.Run(rawOpts); err != nil {
		return nil, fmt.Errorf("error running CRD generator: %w", err)
	}
	cache := gen.GetCache()
	infos, err := afero.ReadDir(cache, cacheOutputDir)
	if err != nil {
		return nil, fmt.Errorf("error reading CRD cache dir %s: %w", cacheOutputDir, err)
	}
	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		path := filepath.Join(cacheOutputDir, info.Name())
		b, err := afero.ReadFile(cache, path)
		if err != nil {
			return nil, fmt.Errorf("error reading cached CRD file %s: %w", path, err)
		}
		scanner := yamlutil.NewYAMLScanner(b)
		modifiedCRD := []byte{}
		for scanner.Scan() {
			crd := unstructured.Unstructured{}
			if err = yaml.Unmarshal(scanner.Bytes(), &crd); err != nil {
				return nil, fmt.Errorf("error unmarshalling CRD manifest %s: %w", path, err)
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
			annotations := crd.GetAnnotations()
			delete(annotations, "controller-gen.kubebuilder.io/version")
			if len(annotations) == 0 {
				annotations = nil
			}
			crd.SetAnnotations(annotations)
			b, err := k8sutil.GetObjectBytes(&crd, yaml.Marshal)
			if err != nil {
				return nil, fmt.Errorf("error marshalling CRD %s: %w", crd.GetName(), err)
			}
			modifiedCRD = yamlutil.CombineManifests(modifiedCRD, b)
		}
		if err = scanner.Err(); err != nil {
			return nil, fmt.Errorf("error scanning CRD manifest %s: %w", path, err)
		}
		if len(modifiedCRD) != 0 {
			fileNameNoExt := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
			fileMap[fileNameNoExt+"_crd.yaml"] = modifiedCRD
		}
	}
	if len(fileMap) == 0 {
		return nil, errors.New("no generated CRD files found")
	}
	return fileMap, nil
}

// generateNonGo generates a CRD for non-Go projects using a resource.
func (g crdGenerator) generateNonGo() (map[string][]byte, error) {
	crd := apiextv1beta1.CustomResourceDefinition{}
	fileMap := map[string][]byte{}
	fileName := getFileNameForResource(g.resource)
	path := filepath.Join(g.Inputs[CRDsDirKey], fileName)
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error stating CRD file %s: %w", path, err)
		}
		crd = newCRDForResource(g.resource)
	} else {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("error reading CRD file %s: %w", path, err)
		}
		if err = yaml.Unmarshal(b, &crd); err != nil {
			return nil, fmt.Errorf("error unmarshalling CRD manifest %s: %w", path, err)
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
	}

	sort.Sort(k8sutil.CRDVersions(crd.Spec.Versions))
	setCRDStorageVersion(&crd)
	if err := checkCRDVersions(crd); err != nil {
		return nil, fmt.Errorf("error checking CRD %s versions: %w", crd.GetName(), err)
	}
	b, err := k8sutil.GetObjectBytes(&crd, yaml.Marshal)
	if err != nil {
		return nil, fmt.Errorf("error marshalling CRD %s: %w", crd.GetName(), err)
	}
	fileMap[fileName] = b
	return fileMap, nil
}

// newCRDForResource constructs a barebones CRD using data in resource.
func newCRDForResource(r scaffold.Resource) apiextv1beta1.CustomResourceDefinition {
	trueVal := true
	return apiextv1beta1.CustomResourceDefinition{
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
			Validation: &apiextv1beta1.CustomResourceValidation{
				OpenAPIV3Schema: &apiextv1beta1.JSONSchemaProps{
					Type:                   "object",
					XPreserveUnknownFields: &trueVal,
				},
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
func checkCRDVersions(crd apiextv1beta1.CustomResourceDefinition) error {
	singleVer := crd.Spec.Version != ""
	multiVers := len(crd.Spec.Versions) > 0
	if singleVer {
		if !multiVers {
			log.Warnf("CRD %s: spec.version is deprecated and should be migrated to spec.versions", crd.Spec.Names.Kind)
		} else if crd.Spec.Version != crd.Spec.Versions[0].Name {
			return fmt.Errorf("spec.version %s must be the first element in spec.versions for CRD %s", crd.Spec.Version, crd.Spec.Names.Kind)
		}
	}

	var hasStorageVer bool
	for _, ver := range crd.Spec.Versions {
		// There must be exactly one version flagged as a storage version.
		if ver.Storage {
			if hasStorageVer {
				return fmt.Errorf("%s CRD has more than one storage version", crd.GetName())
			}
			hasStorageVer = true
		}
	}
	if multiVers && !hasStorageVer {
		return fmt.Errorf("%s CRD has no storage version", crd.GetName())
	}
	return nil
}
