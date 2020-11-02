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
	"strings"

	"sigs.k8s.io/controller-tools/pkg/markers"
)

// getMarkedChildrenOfField collects all marked fields from type declarations starting at root in depth-first order.
func (g generator) getMarkedChildrenOfField(root markers.FieldInfo) (map[string][]*fieldInfo, error) {
	// ast.Inspect will not traverse into fields, so iteratively collect them and to check for markers.
	nextFields := []*fieldInfo{{FieldInfo: root}}
	markedFields := map[string][]*fieldInfo{}
	for len(nextFields) > 0 {
		fields := []*fieldInfo{}
		for _, field := range nextFields {
			errs := []error{}
			ast.Inspect(field.RawField, func(n ast.Node) bool {
				if n == nil {
					return true
				}
				switch expr := n.(type) {
				case *ast.Ident:
					// Only look at type names.
					if expr.Obj == nil || expr.Obj.Kind != ast.Typ {
						return true
					}
					// Check if the field's type exists in the known types.
					info, hasInfo := g.types[expr.Name]
					if !hasInfo {
						return true
					}
					// Add all child fields to the list to search next.
					for _, finfo := range info.Fields {
						segment, err := getPathSegmentForField(finfo)
						if err != nil {
							errs = append(errs, fmt.Errorf("error getting path from type %s field %s: %v",
								info.Name, finfo.Name, err),
							)
							return true
						}
						// Add extra information to the segment if it comes from a certain field type.
						switch finfo.RawField.Type.(type) {
						case (*ast.ArrayType):
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
