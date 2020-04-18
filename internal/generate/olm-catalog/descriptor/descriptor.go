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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
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

	// Check if apisDir exists
	exists, err := isDirExist(apisDir)
	if err != nil {
		return olmapiv1alpha1.CRDDescription{}, err
	}
	if !exists {
		log.Debugf("Could not find API types directory: %s", apisDir)
		return olmapiv1alpha1.CRDDescription{}, ErrAPIDirNotExist
	}

	// Check if the kind pkg is at the expected layout
	// multi-group layout: <api-dir>/<group>/<version>
	// single-group layout: <api-dir>/<version>
	expectedPkgPath, err := getExpectedPkgLayout(apisDir, group, gvk.Version)
	if err != nil {
		return olmapiv1alpha1.CRDDescription{}, err
	}

	// Get pkg types for the given GVK
	var pkgTypes []*types.Type
	if expectedPkgPath != "" {
		// Look for the pkg types at the expected single or multi group import path
		universe, err := getPkgsFromDirRecursive(expectedPkgPath)
		if err != nil {
			return olmapiv1alpha1.CRDDescription{}, err
		}
		pkgTypes, err = getTypesForPkgPath(expectedPkgPath, universe)
		if err != nil {
			return olmapiv1alpha1.CRDDescription{}, err
		}
	} else {
		// Unknown apis directory layout: <apis-dir>/.../<version>
		// Look in <api-dir> recursively for expected pkg name <version>

		// TODO: gengo.parse.AddDirRecursive() will (sometimes?) fail if the
		// root apisDir has no .go files.
		// Workaround for this is to have a doc.go file in the package.
		// Move away from using gengo in the future if possible.
		universe, err := getPkgsFromDirRecursive(apisDir)
		if err != nil {
			return olmapiv1alpha1.CRDDescription{}, err
		}
		pkgTypes, err = getTypesForPkgName(gvk.Version, universe)
		if err != nil {
			return olmapiv1alpha1.CRDDescription{}, err
		}
	}

	kindType := findKindType(gvk.Kind, pkgTypes)
	if kindType == nil {
		return olmapiv1alpha1.CRDDescription{}, ErrAPITypeNotFound
	}
	comments := append(kindType.SecondClosestCommentLines, kindType.CommentLines...)
	kindDescriptors, err := parseCSVGenAnnotations(comments)
	if err != nil {
		return olmapiv1alpha1.CRDDescription{}, fmt.Errorf("error parsing CSV type %s annotations: %v",
			kindType.Name.Name, err)
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
			return olmapiv1alpha1.CRDDescription{}, fmt.Errorf("error parsing %s type member %s JSON tags: %v",
				gvk.Kind, member.Name, err)
		}
		if path != typeSpec && path != typeStatus {
			continue
		}
		tree, err := newTypeTreeFromRoot(member.Type)
		if err != nil {
			return olmapiv1alpha1.CRDDescription{}, fmt.Errorf("error creating type tree for member type %s: %v",
				member.Type.Name, err)
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

// getExpectedPkgLayout checks the directory layout in apisDir
// for single and multi group layouts and returns the expected pkg path
// for the group and version.
// Returns empty string if neither single or multi group layout is detected
// multi group path: <api-dir>/<group>/<version>
// single group path: <api-dir>/<version>
func getExpectedPkgLayout(apisDir, group, version string) (expectedPkgPath string, err error) {
	groupVersionDir := filepath.Join(apisDir, group, version)
	if isMultiGroupLayout, err := isDirExist(groupVersionDir); isMultiGroupLayout {
		if err != nil {
			return "", err
		}
		return groupVersionDir, nil
	}
	versionDir := filepath.Join(apisDir, version)
	if isSingleGroupLayout, err := isDirExist(versionDir); isSingleGroupLayout {
		if err != nil {
			return "", err
		}
		return versionDir, nil
	}
	// Neither multi nor single group layout
	return "", nil
}

// getPkgsFromDirRecursive gets all Go types from dir and recursively its sub directories.
// dir must be the project relative path to the pkg directory
func getPkgsFromDirRecursive(dir string) (types.Universe, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}
	p := parser.New()
	// Gengo's AddDirRecursive fails to load subdir pkgs if the root dir
	// isn't the full pkg import path, or begins with ./
	// Use path relative to current dir
	// TODO: Turn abs path into ./... relative path as well
	if !filepath.IsAbs(dir) && !strings.HasPrefix(dir, ".") {
		dir = fmt.Sprintf(".%s%s", string(filepath.Separator), dir)
	}
	// TODO(hasbro17): AddDirRecursive can be noisy with klog warnings
	// when it skips directories with no .go files.
	// Silence those warnings unless in debug mode.
	if err := p.AddDirRecursive(dir); err != nil {
		return nil, err
	}
	universe, err := p.FindTypes()
	if err != nil {
		return nil, err
	}
	return universe, nil
}

// getTypesForPkgPath find the pkg with the given path in universe
func getTypesForPkgPath(pkgPath string, universe types.Universe) (pkgTypes []*types.Type, err error) {
	var pkg *types.Package
	for _, upkg := range universe {
		if strings.HasSuffix(upkg.Path, pkgPath) {
			pkg = upkg
			break
		}
	}
	if pkg == nil {
		return nil, fmt.Errorf("no package found for API %s", pkgPath)
	}
	for _, t := range pkg.Types {
		pkgTypes = append(pkgTypes, t)
	}
	return pkgTypes, nil
}

func getTypesForPkgName(pkgName string, universe types.Universe) (pkgTypes []*types.Type, err error) {
	var pkg *types.Package
	for _, upkg := range universe {
		if upkg.Name == pkgName {
			pkg = upkg
			break
		}
	}
	if pkg == nil {
		return nil, fmt.Errorf("no package found for %s", pkgName)
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
