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
	"fmt"
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
	group := gvk.Group
	if strings.Contains(group, ".") {
		group = strings.Split(group, ".")[0]
	}
	apiDir := filepath.Join(apisDir, group, gvk.Version)
	universe, err := getTypesFromDir(apiDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Infof("API directory %s does not exist. Skipping CSV annotation parsing for API %s.", apiDir, gvk)
			return nil
		}
		return err
	}
	apiPkg := path.Join(projutil.GetGoPkg(), filepath.ToSlash(apiDir))
	specType, statusType, pkgTypes, err := getSpecStatusPkgTypesForAPI(apiPkg, gvk.Kind, universe)
	if err != nil {
		return errors.Wrapf(err, "failed to parse spec, status, and package types for %s", gvk)
	}

	var descriptors []descriptor
	for _, t := range pkgTypes {
		switch t.Kind {
		case types.Struct:
			if t.Name.Name == gvk.Kind {
				comments := append(t.SecondClosestCommentLines, t.CommentLines...)
				pd, err := parseCSVGenAnnotations(comments)
				if err != nil {
					return errors.Wrapf(err, "error parsing CSV type %s annotations", t.Name.Name)
				}
				crdDesc.Description = parseDescription(comments)
				crdDesc.DisplayName = pd.displayName
				crdDesc.Resources = append(crdDesc.Resources, pd.resources...)
			}
			for _, m := range t.Members {
				pd, err := parseCSVGenAnnotations(m.CommentLines)
				if err != nil {
					return errors.Wrapf(err, "error parsing CSV type %s member %s annotations", t.Name.Name, m.Name)
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
				XDescriptors: d.xdescs,
			})
		case typeStatus:
			crdDesc.StatusDescriptors = append(crdDesc.StatusDescriptors, olmapiv1alpha1.StatusDescriptor{
				Description:  d.description,
				DisplayName:  d.displayName,
				Path:         d.path,
				XDescriptors: d.xdescs,
			})
		}
	}
	return nil
}

// getTypesFromDir gets all Go types from dir.
func getTypesFromDir(dir string) (types.Universe, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(dir) && !strings.HasPrefix(dir, ".") {
		dir = fmt.Sprintf("./%s", dir)
	}
	p := parser.New()
	if err := p.AddDirRecursive(dir); err != nil {
		return nil, err
	}
	universe, err := p.FindTypes()
	if err != nil {
		return nil, err
	}
	return universe, nil
}

// getSpecStatusPkgTypesForAPI finds and returns types {kind}Spec, {kind}Status,
// and all types in pkg.
func getSpecStatusPkgTypesForAPI(pkg, kind string, universe types.Universe) (spec, status *types.Type, pkgTypes []*types.Type, err error) {
	for _, upkg := range universe {
		if upkg.Path == "" || !strings.HasPrefix(upkg.Path, pkg) {
			continue
		}
		for _, t := range upkg.Types {
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
		return nil, nil, nil, errors.Errorf("no package types found in API %s", pkg)
	}
	return spec, status, pkgTypes, nil
}
