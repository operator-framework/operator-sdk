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

package descriptor

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/gengo/parser"
	"k8s.io/gengo/types"
)

// GetCRDDescriptorForGVK parses type and struct field declaration comments on
// API types to populate a csv's spec.customresourcedefinitions.owned fields
// for a given API identified by Group, Version, and Kind in apisDir.
// TODO(estroz): support ActionDescriptors parsing/setting.
func GetCRDDescriptorForGVK(apisDir string, crdDesc *olmapiv1alpha1.CRDDescription, gvk schema.GroupVersionKind) error {
	specType, statusType, pkgTypes, found, err := findTypesForGVK(apisDir, gvk)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	var descriptors []descriptor
	for _, t := range pkgTypes {
		switch t.Kind {
		case types.Struct:
			if t.Name.Name == gvk.Kind {
				comments := append(t.SecondClosestCommentLines, t.CommentLines...)
				pd, err := parseCSVGenAnnotations(comments)
				if err != nil {
					return err
				}
				crdDesc.Description = parseDescription(comments)
				crdDesc.DisplayName = pd.displayName
				crdDesc.Resources = append(crdDesc.Resources, pd.resources...)
			}
			for _, m := range t.Members {
				pd, err := parseCSVGenAnnotations(m.CommentLines)
				if err != nil {
					return err
				}
				for _, d := range pd.descriptors {
					d.parentType, d.member = t, m
					descriptors = append(descriptors, d)
				}
			}
		}
	}

	crdDesc.Resources = sortResources(crdDesc.Resources)
	descriptors = mergeChildDescriptorPaths(specType, statusType, descriptors)
	// Now that we've merged child paths, ensure all fields not set are added.
	for i := 0; i < len(descriptors); i++ {
		setDescriptorDefaultsIfEmpty(&descriptors[i])
	}
	for _, d := range sortDescriptors(descriptors) {
		switch d.descType {
		case typeSpec:
			crdDesc.SpecDescriptors = append(crdDesc.SpecDescriptors, olmapiv1alpha1.SpecDescriptor{
				Description:  d.description,
				DisplayName:  d.displayName,
				Path:         d.path,
				XDescriptors: d.xdesc,
			})
		case typeStatus:
			crdDesc.StatusDescriptors = append(crdDesc.StatusDescriptors, olmapiv1alpha1.StatusDescriptor{
				Description:  d.description,
				DisplayName:  d.displayName,
				Path:         d.path,
				XDescriptors: d.xdesc,
			})
		}
	}
	return nil
}

func findTypesForGVK(apisDir string, gvk schema.GroupVersionKind) (*types.Type, *types.Type, []*types.Type, bool, error) {
	group := gvk.Group
	if strings.Contains(group, ".") {
		group = strings.Split(group, ".")[0]
	}
	apiDir := filepath.Join(apisDir, group, gvk.Version)
	universe, err := getTypesFromDir(apiDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Infof("API directory %s does not exist. Skipping CSV annotation parsing for API %s.", apiDir, gvk)
			return nil, nil, nil, false, nil
		}
		return nil, nil, nil, false, err
	}
	apiPkg := path.Join(projutil.GetGoPkg(), filepath.ToSlash(apiDir))
	spec, status, pkgTypes, err := getSpecStatusPkgTypesForAPI(universe, apiPkg, gvk.Kind)
	if err != nil {
		return nil, nil, nil, false, errors.Wrapf(err, "failed to parse spec, status, and package types for %s", gvk)
	}
	return spec, status, pkgTypes, true, nil
}

func getTypesFromDir(apiDir string) (types.Universe, error) {
	if _, err := os.Stat(apiDir); err != nil {
		return nil, err
	}
	p := parser.New()
	if err := p.AddDirRecursive("./" + apiDir); err != nil {
		return nil, err
	}
	universe, err := p.FindTypes()
	if err != nil {
		return nil, err
	}
	return universe, nil
}

// getSpecStatusPkgTypesForAPI finds and returns types {kind}Spec, {kind}Status,
// and all types in apiPkg.
func getSpecStatusPkgTypesForAPI(universe types.Universe, apiPkg, kind string) (spec, status *types.Type, pkgTypes []*types.Type, err error) {
	for _, pkg := range universe {
		if pkg.Path != apiPkg && !strings.HasPrefix(pkg.Path, "./") {
			continue
		}
		for _, t := range pkg.Types {
			pkgTypes = append(pkgTypes, t)
			if t.Name.Name == kind {
				for _, m := range t.Members {
					path := parsePathFromJSONTags(m.Tags)
					if path == typeSpec {
						spec = m.Type
					} else if path == typeStatus {
						status = m.Type
					}
					if spec != nil && status != nil {
						break
					}
				}
			}
		}
	}
	if spec == nil {
		return nil, nil, nil, errors.Errorf("no spec found in type %s", kind)
	}
	if status == nil {
		return nil, nil, nil, errors.Errorf("no status found in type %s", kind)
	}
	if len(pkgTypes) == 0 {
		return nil, nil, nil, errors.Errorf("no package types found in API %s", apiPkg)
	}
	return spec, status, pkgTypes, nil
}
