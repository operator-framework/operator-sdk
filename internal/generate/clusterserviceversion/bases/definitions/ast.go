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
	"strings"

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
		return imports[0].path
	}

	// If multiple files contain the same local import name, check to see which file contains the selector expression.
	for _, imp := range imports {
		if imp.f.Pos() <= expr.Pos() && imp.f.End() >= expr.End() {
			return imp.path
		}
	}
	return ""
}

// getMarkedChildrenOfField collects all marked fields from type declarations starting at rootField in depth-first order.
func (g generator) getMarkedChildrenOfField(rootPkg *loader.Package, rootField markers.FieldInfo) (map[string][]*fieldInfo, error) {
	// Gather all types and imports needed to build the BFS tree.
	rootPkg.NeedTypesInfo()
	importIDs, err := newImportIdents(rootPkg)
	if err != nil {
		return nil, err
	}

	// ast.Inspect will not traverse into fields, so iteratively collect them and to check for markers.
	nextFields := []*fieldInfo{{FieldInfo: rootField}}
	markedFields := map[string][]*fieldInfo{}
	for len(nextFields) > 0 {
		fields := []*fieldInfo{}
		for _, field := range nextFields {
			errs := []error{}
			ast.Inspect(field.RawField, func(n ast.Node) bool {
				if n == nil {
					return true
				}

				var info *markers.TypeInfo
				var hasInfo bool
				switch nt := n.(type) {
				case *ast.SelectorExpr:
					// Case of a type definition in an imported package.

					pkgPath := importIDs.findPackagePathForSelExpr(nt)
					if pkgPath == "" {
						// Found no reference to pkgPath in any file.
						return true
					}
					if pkg, hasImport := rootPkg.Imports()[loader.NonVendorPath(pkgPath)]; hasImport {
						// Check if the field's type exists in the known types.
						info, hasInfo = g.types[crd.TypeIdent{Package: pkg, Name: nt.Sel.Name}]
					}
				case *ast.Ident:
					// Case of a local type definition.

					// Only look at type names.
					if nt.Obj != nil && nt.Obj.Kind == ast.Typ {
						// Check if the field's type exists in the known types.
						info, hasInfo = g.types[crd.TypeIdent{Package: rootPkg, Name: nt.Name}]
					}
				}
				if !hasInfo {
					return true
				}

				// Add all child fields to the list to search next.
				for _, finfo := range info.Fields {
					segment, err := getPathSegmentForField(finfo)
					if err != nil {
						errs = append(errs, fmt.Errorf("error getting path from type %s field %s: %v", info.Name, finfo.Name, err))
						return true
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
						markedFields[info.Name] = append(markedFields[info.Name], f)
					}
				}

				return true
			})
			if err := fmtParseErrors(errs); err != nil {
				return nil, err
			}
		}
		nextFields = fields
	}
	return markedFields, nil
}

// fmtParseErrors prettifies a list of errors to make them easier to read.
func fmtParseErrors(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	}
	sb := strings.Builder{}
	for _, err := range errs {
		sb.WriteString("\n")
		sb.WriteString(err.Error())
	}
	return errors.New(sb.String())
}
