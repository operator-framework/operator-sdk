// Copyright 2020 The Operator-SDK Authors
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

package definitions

import (
	"errors"
	"os"
	"path/filepath"
	"sort"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"golang.org/x/tools/go/packages"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

type descriptionValues struct {
	crdOrder int
	crd      v1alpha1.CRDDescription
	// TODO(estroz): support apiServiceDescriptions
}

// ApplyDefinitionsForKeysGo collects markers and AST info on Go type declarations and struct fields
// to populate csv spec fields. Go code with relevant markers and information is expected to be
// in a package under apisRootDir and match a GVK in keys.
func ApplyDefinitionsForKeysGo(csv *v1alpha1.ClusterServiceVersion, apisRootDir string, gvks []schema.GroupVersionKind) error {

	// Construct a set of probable paths under apisRootDir for types defined by gvks.
	// These are usually '(pkg/)?apis/(<group>/)?<version>'.
	// NB(estroz): using "leaf" packages prevents type builders from searching other packages.
	// It would be nice to implement extra-package traversal in the future.
	paths, err := makeAPIPaths(apisRootDir, gvks)
	if err != nil {
		return err
	}
	// Some APIs may not exist under apisRootDir, so skip loading packages if no paths are found
	if len(paths) == 0 {
		return nil
	}

	// Collect Go types from roots.
	g := &generator{}
	ctx, err := g.contextForRoots(paths...)
	if err != nil {
		return err
	}
	g.needTypes(ctx)
	if loader.PrintErrors(ctx.Roots, packages.TypeError) {
		return errors.New("one or more API packages had type errors")
	}

	// Create definitions for kind types found under the collected roots.
	definitionsByGVK := make(map[schema.GroupVersionKind]*descriptionValues)
	for _, gvk := range gvks {
		kindType, hasKind := g.types[gvk.Kind]
		if !hasKind {
			log.Warnf("Skipping CSV annotation parsing for API %s: type %s not found", gvk, gvk.Kind)
			continue
		}
		crd, crdOrder, err := g.buildCRDDescriptionFromType(gvk, kindType)
		if err != nil {
			return err
		}
		definitionsByGVK[gvk] = &descriptionValues{
			crdOrder: crdOrder,
			crd:      crd,
		}
	}

	// Update csv with all values parsed.
	updateDefinitionsByKey(csv, definitionsByGVK)

	return nil
}

// makeAPIPaths creates a set of API directory paths with apisRootDir as their parent.
func makeAPIPaths(apisRootDir string, gvks []schema.GroupVersionKind) (paths []string, err error) {
	if apisRootDir, err = filepath.Abs(apisRootDir); err != nil {
		return nil, err
	}

	for _, gvk := range gvks {
		// Check if the kind pkg is at the expected layout.
		group := MakeGroupFromFullGroup(gvk.Group)
		expectedPkgPath, err := getExpectedPkgLayout(apisRootDir, group, gvk.Version)
		if err != nil {
			return nil, err
		}
		if expectedPkgPath == "" {
			log.Warnf("Skipping CSV annotation parsing for API %s: directory does not exist", gvk)
			continue
		}
		paths = append(paths, expectedPkgPath)
	}
	return paths, nil
}

// updateDefinitionsByKey updates owned definitions that already exist in csv or adds new definitions that do not.
func updateDefinitionsByKey(csv *v1alpha1.ClusterServiceVersion, defsByGVK map[schema.GroupVersionKind]*descriptionValues) {
	// Create a set of buckets for all generated descriptions.
	// Multiple descriptions can belong to the same order.
	crdBuckets := make(map[int][]v1alpha1.CRDDescription)
	for _, values := range defsByGVK {
		crdBuckets[values.crdOrder] = append(crdBuckets[values.crdOrder], values.crd)
	}

	// Sort generated buckets before adding non-generated descriptions so users can
	// set their order manually.
	for _, bucket := range crdBuckets {
		sort.Slice(bucket, func(i, j int) bool {
			return bucket[i].Name < bucket[j].Name
		})
	}

	// Append non-generated descriptions to the end of their buckets,
	// treating their indices as order.
	for i, crd := range csv.Spec.CustomResourceDefinitions.Owned {
		if _, hasKey := defsByGVK[descToGVK(crd)]; !hasKey {
			crdBuckets[i] = append(crdBuckets[i], csv.Spec.CustomResourceDefinitions.Owned[i])
		}
	}

	// De-duplciate and sort order ints for appending bucket contents in-order.
	crdOrders := make([]int, 0, len(crdBuckets))
	for order := range crdBuckets {
		crdOrders = append(crdOrders, order)
	}
	sort.Ints(crdOrders)

	// Update descriptions.
	csv.Spec.CustomResourceDefinitions.Owned = make([]v1alpha1.CRDDescription, 0, len(crdBuckets))
	for _, order := range crdOrders {
		csv.Spec.CustomResourceDefinitions.Owned = append(csv.Spec.CustomResourceDefinitions.Owned, crdBuckets[order]...)
	}
}

// descToGVK convert desc to a GVK type.
func descToGVK(desc v1alpha1.CRDDescription) (gvk schema.GroupVersionKind) {
	gvk.Group = MakeFullGroupFromName(desc.Name)
	gvk.Version = desc.Version
	gvk.Kind = desc.Kind
	return gvk
}

func isDirExist(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return fileInfo.IsDir(), nil
}

// getExpectedPkgLayout checks the directory layout in apisRootDir for single and multi group layouts and returns
// the expected pkg path for the group and version. Returns empty string if neither single or multi group layout
// is detected.
// - multi-group layout:  apis/<group>/<version>
// - single-group layout: api/<version>
func getExpectedPkgLayout(apisRootDir, group, version string) (expectedPkgPath string, err error) {
	if group == "" || version == "" {
		return "", nil
	}
	groupVersionDir := filepath.Join(apisRootDir, group, version)
	if isMultiGroupLayout, err := isDirExist(groupVersionDir); isMultiGroupLayout {
		if err != nil {
			return "", err
		}
		return groupVersionDir, nil
	}
	versionDir := filepath.Join(apisRootDir, version)
	if isSingleGroupLayout, err := isDirExist(versionDir); isSingleGroupLayout {
		if err != nil {
			return "", err
		}
		return versionDir, nil
	}
	// Neither multi nor single group layout
	return "", nil
}

// generator creates API definitions from type information for a set of roots.
type generator struct {
	types map[string]*markers.TypeInfo
}

// contextForRoots creates a context that can populate a generator for a set of roots loaded from dirs.
// These roots contain data for registered markers.
func (g *generator) contextForRoots(dirs ...string) (ctx *genall.GenerationContext, err error) {
	roots, err := loader.LoadRoots(dirs...)
	if err != nil {
		return ctx, err
	}
	registry := &markers.Registry{}
	if err := registerMarkers(registry); err != nil {
		return ctx, err
	}
	return &genall.GenerationContext{
		Collector: &markers.Collector{
			Registry: registry,
		},
		Roots:     roots,
		InputRule: genall.InputFromFileSystem,
		Checker:   &loader.TypeChecker{},
	}, nil
}

// needTypes sets types in the generator for a given context.
func (g *generator) needTypes(ctx *genall.GenerationContext) {
	g.types = make(map[string]*markers.TypeInfo)
	cb := func(info *markers.TypeInfo) {
		g.types[info.Name] = info
	}
	for _, root := range ctx.Roots {
		if err := markers.EachType(ctx.Collector, root, cb); err != nil {
			root.AddError(err)
		}
	}
}
