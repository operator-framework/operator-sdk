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
	pkgTypes, err := getTypesForPkg(apiPkg, universe)
	if err != nil {
		return err
	}
	kindType := findKindType(gvk.Kind, pkgTypes)
	if kindType == nil {
		log.Infof("No type %s found. Skipping CSV annotation parsing for API %s.", gvk.Kind, gvk)
		return nil
	}
	comments := append(kindType.SecondClosestCommentLines, kindType.CommentLines...)
	kindDescriptors, err := parseCSVGenAnnotations(comments)
	if err != nil {
		return errors.Wrapf(err, "error parsing CSV type %s annotations", kindType.Name.Name)
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

	tree := newTypeTreeFromRoot(kindType)
	descriptors, err := tree.getDescriptors()
	if err != nil {
		return err
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

func getTypesForPkg(pkgPath string, universe types.Universe) (pkgTypes []*types.Type, err error) {
	var pkg *types.Package
	for _, upkg := range universe {
		if strings.HasPrefix(upkg.Path, pkgPath) {
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
