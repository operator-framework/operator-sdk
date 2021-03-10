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
	apiversion "k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/controller-tools/pkg/crd"
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
	// Skip definitions parsing if dir doesn't exist, otherwise g.contextForRoots() will error.
	if _, err := os.Stat(apisRootDir); err != nil && errors.Is(err, os.ErrNotExist) {
		log.Warnf("Skipping definitions parsing: APIs root dir %q does not exist", apisRootDir)
		return nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Create a generator context for type-checking and loading all packages under apisRootDir.
	// The "$(pwd)/<package>/..." syntax directs the loader to load all packages under apisRootDir.
	g := &generator{}
	ctx, err := g.contextForRoots(filepath.Join(wd, apisRootDir) + "/...")
	if err != nil {
		return err
	}
	// Collect Go types from the API root.
	g.needTypes(ctx)
	if loader.PrintErrors(ctx.Roots, packages.TypeError) {
		return errors.New("one or more API packages had type errors")
	}

	gvkSet := make(map[schema.GroupVersionKind]struct{}, len(gvks))
	for _, gvk := range gvks {
		gvkSet[gvk] = struct{}{}
	}

	// Create definitions for kind types found under the collected roots.
	definitionsByGVK := make(map[schema.GroupVersionKind]*descriptionValues)
	for typeIdent, typeInfo := range g.types {
		if gv, hasGV := g.groupVersions[typeIdent.Package]; hasGV {
			gvk := gv.WithKind(typeIdent.Name)
			// Type is one of the GVKs specified by the caller.
			if _, hasGVK := gvkSet[gvk]; hasGVK {
				crd, crdOrder, err := g.buildCRDDescriptionFromType(gvk, typeIdent, typeInfo)
				if err != nil {
					return err
				}
				definitionsByGVK[gvk] = &descriptionValues{
					crdOrder: crdOrder,
					crd:      crd,
				}
				delete(gvkSet, gvk)
			}
		}
	}

	// Leftover GVKs are ignored because their types can't be found.
	for _, gvk := range gvkSet {
		log.Warnf("Skipping definitions parsing for API %s: Go type not found", gvk)
	}

	// Update csv with all values parsed.
	updateDefinitionsByKey(csv, definitionsByGVK)

	return nil
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
		sort.Slice(bucket, lessForCRDDescription(bucket))
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

// lessForCRDDescription returns a less func for descs. Used for sorting a list of CRDDescriptions.
func lessForCRDDescription(descs []v1alpha1.CRDDescription) func(i, j int) bool {
	return func(i, j int) bool {
		if descs[i].Name == descs[j].Name {
			if descs[i].Kind == descs[j].Kind {
				return apiversion.CompareKubeAwareVersionStrings(descs[i].Version, descs[j].Version) > 0
			}
			return descs[i].Kind < descs[j].Kind
		}
		return descs[i].Name < descs[j].Name
	}
}

// descToGVK convert desc to a GVK type.
func descToGVK(desc v1alpha1.CRDDescription) (gvk schema.GroupVersionKind) {
	gvk.Group = MakeFullGroupFromName(desc.Name)
	gvk.Version = desc.Version
	gvk.Kind = desc.Kind
	return gvk
}

// generator creates API definitions from type information for a set of roots.
type generator struct {
	types         map[crd.TypeIdent]*markers.TypeInfo
	groupVersions map[*loader.Package]schema.GroupVersion
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
// Adapted from https://github.com/kubernetes-sigs/controller-tools/blob/868d39a/pkg/crd/parser.go#L121
func (g *generator) needTypes(ctx *genall.GenerationContext) {
	g.types = make(map[crd.TypeIdent]*markers.TypeInfo)
	g.groupVersions = make(map[*loader.Package]schema.GroupVersion)
	for _, root := range ctx.Roots {
		pkgMarkers, err := markers.PackageMarkers(ctx.Collector, root)
		if err != nil {
			root.AddError(err)
		} else {
			// Explicitly skip this package.
			if skipPkg := pkgMarkers.Get("kubebuilder:skip"); skipPkg != nil {
				return
			}
			// Get group name and optionall version name from package markers.
			if nameVal := pkgMarkers.Get("groupName"); nameVal != nil {
				versionVal := root.Name
				if versionMarker := pkgMarkers.Get("versionName"); versionMarker != nil {
					versionVal = versionMarker.(string)
				}

				g.groupVersions[root] = schema.GroupVersion{
					Version: versionVal,
					Group:   nameVal.(string),
				}
			}
		}
		// Add all types indexed by their package and type name.
		f := func(info *markers.TypeInfo) {
			ident := crd.TypeIdent{
				Package: root,
				Name:    info.Name,
			}
			g.types[ident] = info
		}
		if err := markers.EachType(ctx.Collector, root, f); err != nil {
			root.AddError(err)
		}
	}
}
