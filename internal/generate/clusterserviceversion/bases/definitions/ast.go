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
	"fmt"
	"go/ast"
	"strconv"

	"golang.org/x/tools/go/packages"
	"sigs.k8s.io/controller-tools/pkg/crd"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"
)

// importIdents maps import identifiers to a list of corresponding path and file containing that path.
// For example, consider the set of 2 files, one containing 'import foo "my/foo"' and the other 'import foo "your/foo"'.
// Then the map would be: map["foo"][]struct{{f: (file 1), path: "my/foo"},{f: (file 2), path: "your/foo"}}.
type importIdents map[string][]struct {
	f    *ast.File
	path string
}

// newImportIdents creates an importIdents from all imports in pkg.
func newImportIdents(pkg *loader.Package) (importIdents, error) {
	importIDs := make(map[string][]struct {
		f    *ast.File
		path string
	})
	for _, file := range pkg.Syntax {
		for _, impSpec := range file.Imports {
			val, err := strconv.Unquote(impSpec.Path.Value)
			if err != nil {
				return nil, err
			}
			// Most imports are not locally named, so the real package name should be used.
			var impName string
			if imp, hasImp := pkg.Imports()[val]; hasImp {
				impName = imp.Name
			}
			// impSpec.Name will not be empty for locally named imports
			if impSpec.Name != nil {
				impName = impSpec.Name.Name
			}
			importIDs[impName] = append(importIDs[impName], struct {
				f    *ast.File
				path string
			}{file, val})
		}
	}
	return importIDs, nil
}

// findPackagePathForSelExpr returns the package path corresponding to the package name used in expr if it exists in im.
func (im importIdents) findPackagePathForSelExpr(expr *ast.SelectorExpr) (pkgPath string) {
	// X contains the name being selected from.
	xIdent, isIdent := expr.X.(*ast.Ident)
	if !isIdent {
		return ""
	}
	// Imports for all import statements where local import name == name being selected from.
	imports, hasImports := im[xIdent.String()]
	if !hasImports {
		return ""
	}

	// Short-circuit if only one import.
	if len(imports) == 1 {
		pkgPath = imports[0].path
	} else {
		// If multiple files contain the same local import name, check to see which file contains the selector expression.
		for _, imp := range imports {
			if imp.f.Pos() <= expr.Pos() && imp.f.End() >= expr.End() {
				pkgPath = imp.path
				break
			}
		}
	}
	return loader.NonVendorPath(pkgPath)
}

// getMarkedChildrenOfField collects all marked fields from type declarations starting at rootField in depth-first order.
func (g generator) getMarkedChildrenOfField(rootPkg *loader.Package, rootField markers.FieldInfo) (map[crd.TypeIdent][]*fieldInfo, error) {
	// Gather all types and imports needed to build the BFS tree.
	rootPkg.NeedTypesInfo()
	importIDs, err := newImportIdents(rootPkg)
	if err != nil {
		return nil, err
	}

	// ast.Inspect will not traverse into fields, so iteratively collect them and to check for markers.
	nextFields := []*fieldInfo{{FieldInfo: rootField}}
	markedFields := map[crd.TypeIdent][]*fieldInfo{}
	for len(nextFields) > 0 {
		fields := []*fieldInfo{}
		for _, field := range nextFields {
			ast.Inspect(field.RawField, func(n ast.Node) bool {
				if n == nil {
					return true
				}

				var ident crd.TypeIdent
				switch nt := n.(type) {
				case *ast.SelectorExpr: // Type definition in an imported package.
					pkgPath := importIDs.findPackagePathForSelExpr(nt)
					if pkgPath == "" {
						// Found no reference to pkgPath in any file.
						return true
					}
					if pkg, hasImport := rootPkg.Imports()[pkgPath]; hasImport {
						pkg.NeedTypesInfo()
						ident = crd.TypeIdent{Package: pkg, Name: nt.Sel.Name}
					}
				case *ast.Ident: // Local type definition.
					// Only look at type idents or references to type idents in other files.
					if nt.Obj == nil || nt.Obj.Kind == ast.Typ {
						ident = crd.TypeIdent{Package: rootPkg, Name: nt.Name}
					}
				}

				// Check if the field's type is a known types.
				info, hasInfo := g.types[ident]
				if ident == (crd.TypeIdent{}) || !hasInfo {
					return true
				}

				// Add all child fields to the list to search next.
				for _, finfo := range info.Fields {
					segment, err := getPathSegmentForField(finfo)
					if err != nil {
						rootPkg.AddError(fmt.Errorf("error getting path from type %s field %s: %v", info.Name, finfo.Name, err))
						continue
					}
					// Add extra information to the segment if it comes from a certain field type.
					switch finfo.RawField.Type.(type) {
					case *ast.ArrayType:
						// arrayFieldGroup case.
						if segment != ignoredTag && segment != inlinedTag {
							segment += "[0]"
						}
					}
					// Create a new set of path segments using the parent's segments
					// and add the field to the next fields to search.
					parentSegments := make([]string, len(field.pathSegments), len(field.pathSegments)+1)
					copy(parentSegments, field.pathSegments)
					f := &fieldInfo{
						FieldInfo:    finfo,
						pathSegments: append(parentSegments, segment),
					}
					fields = append(fields, f)
					// Marked fields get collected for the caller to parse.
					if len(finfo.Markers) != 0 {
						markedFields[ident] = append(markedFields[ident], f)
					}
				}

				return true
			})
		}
		nextFields = fields
	}

	if loader.PrintErrors([]*loader.Package{rootPkg}, packages.TypeError) {
		return nil, errors.New("package had type errors")
	}

	return markedFields, nil
}
