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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/gengo/parser"
	"k8s.io/gengo/types"
)

var (
	// ErrAPIDirNotExist is returned if an API directory does not exist.
	ErrAPIDirNotExist = errors.New("directory for API does not exist")
	// ErrAPITypeNotFound is returned if no type with a name matching the kind
	// of an API is found.
	ErrAPITypeNotFound = errors.New("kind type for API not found")
)

// GetCRDDescriptionForGVK parses type and struct field declaration comments on
// API types to populate a csv's spec.customresourcedefinitions.owned fields
// for a given API identified by Group, Version, and Kind in apisDir.
// TODO(estroz): support ActionDescriptors parsing/setting.
func GetCRDDescriptionForGVK(apisDir string, gvk schema.GroupVersionKind) (olmapiv1alpha1.CRDDescription, error) {
	crdDesc := olmapiv1alpha1.CRDDescription{
		Version: gvk.Version,
		Kind:    gvk.Kind,
	}
	group := gvk.Group
	if strings.Contains(group, ".") {
		group = strings.Split(group, ".")[0]
	}
	apiDir := filepath.Join(apisDir, group, gvk.Version)
	universe, err := getTypesFromDir(apiDir)
	if err != nil {
		if os.IsNotExist(err) {
			return olmapiv1alpha1.CRDDescription{}, ErrAPIDirNotExist
		}
		return olmapiv1alpha1.CRDDescription{}, err
	}
	apiPkg := path.Join(projutil.GetGoPkg(), filepath.ToSlash(apiDir))
	pkgTypes, err := getTypesForPkg(apiPkg, universe)
	if err != nil {
		return olmapiv1alpha1.CRDDescription{}, err
	}
	kindType := findKindType(gvk.Kind, pkgTypes)
	if kindType == nil {
		return olmapiv1alpha1.CRDDescription{}, ErrAPITypeNotFound
	}
	comments := append(kindType.SecondClosestCommentLines, kindType.CommentLines...)
	kindDescriptors, err := parseCSVGenAnnotations(comments)
	if err != nil {
		return olmapiv1alpha1.CRDDescription{}, errors.Wrapf(err, "error parsing CSV type %s annotations", kindType.Name.Name)
	}
	if description := parseDescription(comments); description != "" {
		crdDesc.Description = description
	}
	if kindDescriptors.displayName != "" {
		crdDesc.DisplayName = kindDescriptors.displayName
	}
	if len(kindDescriptors.resources) != 0 {
		crdDesc.Resources = sortResources(kindDescriptors.resources)
	}
	for _, member := range kindType.Members {
		path, err := getPathFromMember(member)
		if err != nil {
			return olmapiv1alpha1.CRDDescription{}, errors.Wrapf(err, "error parsing %s type member %s JSON tags", gvk.Kind, member.Name)
		}
		if path != typeSpec && path != typeStatus {
			continue
		}
		tree, err := newTypeTreeFromRoot(member.Type)
		if err != nil {
			return olmapiv1alpha1.CRDDescription{}, errors.Wrapf(err, "error creating type tree for member type %s", member.Type.Name)
		}
		descriptors, err := tree.getDescriptorsFor(path)
		if err != nil {
			return olmapiv1alpha1.CRDDescription{}, err
		}
		if path == typeSpec {
			for _, d := range sortDescriptors(descriptors) {
				crdDesc.SpecDescriptors = append(crdDesc.SpecDescriptors, d.SpecDescriptor)
			}
		} else {
			for _, d := range sortDescriptors(descriptors) {
				crdDesc.StatusDescriptors = append(crdDesc.StatusDescriptors, olmapiv1alpha1.StatusDescriptor{
					Description:  d.Description,
					DisplayName:  d.DisplayName,
					Path:         d.Path,
					XDescriptors: d.XDescriptors,
				})
			}
		}
	}
	return crdDesc, nil
}

// getTypesFromDir gets all Go types from dir.
func getTypesFromDir(dir string) (types.Universe, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(dir) && !strings.HasPrefix(dir, ".") {
		dir = fmt.Sprintf(".%s%s", string(filepath.Separator), dir)
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

func getTypesForPkg(pkgPath string, universe types.Universe) (pkgTypes []*types.Type, err error) {
	var pkg *types.Package
	for _, upkg := range universe {
		if strings.HasPrefix(upkg.Path, pkgPath) || strings.HasPrefix(upkg.Path, "."+string(filepath.Separator)) {
			pkg = upkg
			break
		}
	}
	if pkg == nil {
		return nil, errors.Errorf("no package found for API %s", pkgPath)
	}
	for _, t := range pkg.Types {
		pkgTypes = append(pkgTypes, t)
	}
	return pkgTypes, nil
}

func findKindType(kind string, pkgTypes []*types.Type) *types.Type {
	for _, t := range pkgTypes {
		if t.Name.Name == kind {
			return t
		}
	}
	return nil
}
