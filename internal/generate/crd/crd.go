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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextinstall "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/install"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/yaml"

	"github.com/operator-framework/operator-sdk/internal/generate/gen"
	"github.com/operator-framework/operator-sdk/internal/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/fileutil"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

const DefaultCRDVersion = "v1"

// Generator configures the CustomResourceDefintion manifest generator
// for Go and non-Go projects.
type Generator struct {
	// OperatorName is the operator's name, ex. app-operator
	OperatorName string
	// OutputDir is the root directory where the output files will be generated.
	OutputDir string
	// isOperatorGo is true when the operator is written in Go.
	IsOperatorGo bool
	// resource contains API information used to configure single-CRD generation.
	// This is only required when isOperatorGo is false.
	Resource scaffold.Resource
	// crdVersion is the API version of the CRD that will be generated.
	// Should be one of [v1, v1beta1]
	CRDVersion string
	// CRDsDir is for the location of the CRD manifests directory e.g "deploy/crds"
	// Both the CRD and CR manifests from this path will be used to populate CSV fields
	// metadata.annotations.alm-examples for CR examples
	// and spec.customresourcedefinitions.owned for owned CRDs
	CRDsDir string
	// ApisDir is for the location of the API types directory e.g "pkg/apis"
	// The CSV annotation comments will be parsed from the types under this path.
	ApisDir string
}

func (g Generator) validate() error {
	if g.CRDsDir == "" {
		return errors.New("input CRDs dir cannot be empty")
	}
	if g.IsOperatorGo && g.ApisDir == "" {
		return errors.New("input APIs dir cannot be empty")
	}
	if g.OutputDir == "" {
		return errors.New("output dir cannot be empty")
	}
	if !g.IsOperatorGo {
		if err := g.Resource.Validate(); err != nil {
			return fmt.Errorf("resource is invalid: %w", err)
		}
	}
	switch g.CRDVersion {
	case "v1", "v1beta1":
	default:
		return fmt.Errorf("crd version %q is invalid", g.CRDVersion)
	}
	return nil
}

// Generate generates CRD manifests and writes them to g.OutputDir.
func (g Generator) Generate() (err error) {
	if g.CRDsDir == "" {
		g.CRDsDir = scaffold.CRDsDir
	}
	if g.ApisDir == "" {
		g.ApisDir = scaffold.ApisDir
	}
	if g.OutputDir == "" {
		g.OutputDir = g.CRDsDir
	}
	if err = g.validate(); err != nil {
		return fmt.Errorf("error validating generator configuration: %w", err)
	}
	var fileMap map[string][]byte
	if g.IsOperatorGo {
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
func (g Generator) generateGo() (map[string][]byte, error) {
	fileMap := map[string][]byte{}
	// Generate files in the generator's cache so we can modify the file name
	// and annotations.
	defName := "output:crd:cache"
	cacheOutputDir := filepath.Clean(g.OutputDir)
	rawOpts := []string{
		fmt.Sprintf("crd:crdVersions={%s}", g.CRDVersion),
		fmt.Sprintf("paths=%s/...", fileutil.DotPath(g.ApisDir)),
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
		scanner := k8sutil.NewYAMLScanner(bytes.NewBuffer(b))
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
			modifiedCRD = k8sutil.CombineManifests(modifiedCRD, b)
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
func (g Generator) generateNonGo() (map[string][]byte, error) {
	// Since this method is usually only run once, we can initialize this
	// scheme when called.
	scheme := runtime.NewScheme()
	apiextinstall.Install(scheme)
	dec := serializer.NewCodecFactory(scheme).UniversalDeserializer()

	var crd *apiextv1beta1.CustomResourceDefinition
	var preserveUnknownFields *bool
	fileMap := map[string][]byte{}
	fileName := getFileNameForResource(g.Resource)
	path := filepath.Join(g.CRDsDir, fileName)
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error stating CRD file %s: %w", path, err)
		}
		crd = newCRDForResource(g.Resource)
	} else {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("error reading CRD file %s: %w", path, err)
		}

		// Decode the CRD manifest in a GVK-aware manner.
		obj, _, err := dec.Decode(b, nil, nil)
		if err != nil {
			return nil, err
		}

		switch t := obj.(type) {
		case *apiextv1.CustomResourceDefinition:
			if crd, err = convertv1Tov1beta1CustomResourceDefinition(t); err != nil {
				return nil, fmt.Errorf("error converting CustomResourceDefinition v1 to v1beta1: %v", err)
			}
		case *apiextv1beta1.CustomResourceDefinition:
			// Only set spec.preserveUnknownFields if it was set before conversion.
			preserveUnknownFields = t.Spec.PreserveUnknownFields
			crd = t
		default:
			return nil, fmt.Errorf("unrecognized type in CustomResourceDefinition getter: %T", t)
		}

		// If version is new, append it to spec.versions.
		hasVersion := false
		for _, version := range crd.Spec.Versions {
			if version.Name == g.Resource.Version {
				hasVersion = true
				break
			}
		}
		if !hasVersion {
			// Let either the user or below logic determine whether this new
			// version is stored or not.
			crd.Spec.Versions = append(crd.Spec.Versions, apiextv1beta1.CustomResourceDefinitionVersion{
				Name:    g.Resource.Version,
				Storage: false,
				Served:  true,
			})
		}
	}

	sort.Sort(k8sutil.CRDVersions(crd.Spec.Versions))
	setCRDStorageVersion(crd)
	if err := checkCRDVersions(crd); err != nil {
		return nil, fmt.Errorf("invalid version in CRD %s: %w", crd.GetName(), err)
	}

	var (
		b   []byte
		err error
	)
	switch g.CRDVersion {
	case "v1beta1":
		// If converting from v1, this will be nil and handled by a validation key.
		// Otherwise this is a no-op.
		crd.Spec.PreserveUnknownFields = preserveUnknownFields
		b, err = k8sutil.GetObjectBytes(crd, yaml.Marshal)
	case "v1":
		out, cerr := k8sutil.Convertv1beta1Tov1CustomResourceDefinition(crd)
		if cerr != nil {
			return nil, fmt.Errorf("error converting CustomResourceDefinition v1beta1 to v1: %v", err)
		}
		b, err = k8sutil.GetObjectBytes(out, yaml.Marshal)
	}
	if err != nil {
		return nil, fmt.Errorf("error marshalling CRD %s: %w", crd.GetName(), err)
	}

	fileMap[fileName] = b
	return fileMap, nil
}

//nolint:lll
func convertv1Tov1beta1CustomResourceDefinition(in *apiextv1.CustomResourceDefinition) (*apiextv1beta1.CustomResourceDefinition, error) {
	var unversioned apiext.CustomResourceDefinition
	//nolint:lll
	if err := apiextv1.Convert_v1_CustomResourceDefinition_To_apiextensions_CustomResourceDefinition(in, &unversioned, nil); err != nil {
		return nil, err
	}
	var out apiextv1beta1.CustomResourceDefinition
	out.TypeMeta.APIVersion = apiextv1beta1.SchemeGroupVersion.String()
	out.TypeMeta.Kind = "CustomResourceDefinition"
	//nolint:lll
	if err := apiextv1beta1.Convert_apiextensions_CustomResourceDefinition_To_v1beta1_CustomResourceDefinition(&unversioned, &out, nil); err != nil {
		return nil, err
	}
	return &out, nil
}

// newCRDForResource constructs a barebones CRD using data in resource.
func newCRDForResource(r scaffold.Resource) *apiextv1beta1.CustomResourceDefinition {
	trueVal := true
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
func checkCRDVersions(crd *apiextv1beta1.CustomResourceDefinition) error {
	singleVer := crd.Spec.Version != ""
	multiVers := len(crd.Spec.Versions) > 0
	if singleVer {
		if !multiVers {
			log.Warnf("CRD %s: spec.version is deprecated and should be migrated to spec.versions",
				crd.Spec.Names.Kind)
		} else if crd.Spec.Version != crd.Spec.Versions[0].Name {
			return fmt.Errorf("spec.version %s must be the first element in spec.versions for CRD %s",
				crd.Spec.Version, crd.Spec.Names.Kind)
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
