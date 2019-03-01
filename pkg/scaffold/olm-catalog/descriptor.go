// Copyright 2018 The Operator-SDK Authors
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

package catalog

import (
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/scaffold"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type fieldVals struct {
	name, typ, tag, comments string
}

func setCRDDescriptorsForGV(crdDesc *olmapiv1alpha1.CRDDescription, gv schema.GroupVersion) error {
	fset := token.NewFileSet()
	ff := func(info os.FileInfo) bool {
		return strings.HasSuffix(info.Name(), "_types.go")
	}
	dir := filepath.Join(scaffold.ApisDir, strings.Split(gv.Group, ".")[0], gv.Version)
	pkgs, err := parser.ParseDir(fset, dir, ff, parser.ParseComments)
	if err != nil {
		// Don't return in error as other CSV components can still be generated.
		if os.IsNotExist(err) {
			return nil
		}
		log.Fatal(err)
	}

	allTypes := make(map[string][]fieldVals)
	for _, pkg := range pkgs {
		ast.Inspect(pkg, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.GenDecl:
				for _, spec := range x.Specs {
					if spec, ok := spec.(*ast.TypeSpec); ok {
						specName := spec.Name.Name
						if specName == crdDesc.Kind {
							crdDesc.Description = joinComments(x.Doc.Text())
						}
						vals, ok := allTypes[specName]
						if !ok {
							vals = make([]fieldVals, 0)
						}
						switch st := spec.Type.(type) {
						case *ast.StructType:
							for _, field := range st.Fields.List {
								typ := processType(fset, field.Type)
								comments := joinComments(field.Doc.Text())
								names := field.Names
								if len(names) == 0 {
									names = []*ast.Ident{{Name: typ}}
								}
								for _, name := range names {
									// Is exported, primitive, or imported.
									if name.IsExported() || strings.Contains(name.Name, ".") {
										vals = append(vals, fieldVals{
											name:     getDisplayName(name.Name),
											typ:      typ,
											tag:      field.Tag.Value,
											comments: comments,
										})
									}
								}
							}
						}
						allTypes[specName] = vals
					}
				}
			}
			return true
		})
	}
	kindTypes, ok := allTypes[crdDesc.Kind]
	if !ok {
		return fmt.Errorf("no type found for kind %s", crdDesc.Kind)
	}
	for _, kt := range kindTypes {
		for _, fields := range allTypes[kt.typ] {
			if strings.HasSuffix(kt.name, "Spec") {
				path, xd := guessPathAndXDescriptorFromTag(fields.tag, true)
				crdDesc.SpecDescriptors = append(crdDesc.SpecDescriptors, olmapiv1alpha1.SpecDescriptor{
					Description:  fields.comments,
					DisplayName:  fields.name,
					Path:         path,
					XDescriptors: xd,
				})
			} else if strings.HasSuffix(kt.name, "Status") {
				path, xd := guessPathAndXDescriptorFromTag(fields.tag, false)
				crdDesc.StatusDescriptors = append(crdDesc.StatusDescriptors, olmapiv1alpha1.StatusDescriptor{
					Description:  fields.comments,
					DisplayName:  fields.name,
					Path:         path,
					XDescriptors: xd,
				})
			}
		}
	}
	return nil
}

func joinComments(comments string) string {
	lines := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(comments))
	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())
		if l == "" || strings.Contains(l, "+k8s:") || strings.Contains(l, "+kubebuilder:") {
			continue
		}
		lines = append(lines, l)
	}
	if err := scanner.Err(); err != nil {
		log.Error(err)
	}
	return strings.Join(lines, " ")
}

var mapRe = regexp.MustCompile(`map\[.+\]`)

func processType(fset *token.FileSet, e ast.Expr) (t string) {
	tbuf := &bytes.Buffer{}
	if err := printer.Fprint(tbuf, fset, e); err != nil {
		log.Fatal(err)
	}
	t = tbuf.String()
	tt := strings.Replace(t, "[]", "", -1)
	tt = strings.Replace(tt, "*", "", -1)
	tt = mapRe.ReplaceAllString(tt, "")
	// Only remove braces, map, pointer syntax from non-primitive types.
	if types.Universe.Lookup(tt) == nil {
		t = tt
	}
	return t
}

// From https://github.com/openshift/console/blob/master/frontend/public/components/operator-lifecycle-manager/descriptors/types.ts#L5-L14
var specXDescriptors = map[string][]string{
	"size":                 {"size", "urn:alm:descriptor:com.tectonic.ui:podCount"},
	"endpointList":         {"endpointList", "urn:alm:descriptor:com.tectonic.ui:endpointList"},
	"label":                {"label", "urn:alm:descriptor:com.tectonic.ui:label"},
	"resourceRequirements": {"resourceRequirements", "urn:alm:descriptor:com.tectonic.ui:resourceRequirements"},
	"selector":             {"selector", "urn:alm:descriptor:com.tectonic.ui:selector:"},
	"namespaceSelector":    {"namespaceSelector", "urn:alm:descriptor:com.tectonic.ui:namespaceSelector"},
	"booleanSwitch":        {"booleanSwitch", "urn:alm:descriptor:com.tectonic.ui:booleanSwitch"},
}

// From https://github.com/openshift/console/blob/master/frontend/public/components/operator-lifecycle-manager/descriptors/types.ts#L16-L27
var statusXDescriptors = map[string][]string{
	"nodes":              {"size", "urn:alm:descriptor:com.tectonic.ui:podCount"},
	"size":               {"size", "urn:alm:descriptor:com.tectonic.ui:podCount"},
	"podStatuses":        {"podStatuses", "urn:alm:descriptor:com.tectonic.ui:podStatuses"},
	"w3Link":             {"w3Link", "urn:alm:descriptor:org.w3:link"},
	"conditions":         {"conditions", "urn:alm:descriptor:io.kubernetes.conditions"},
	"text":               {"text", "urn:alm:descriptor:text"},
	"prometheusEndpoint": {"prometheusEndpoint", "urn:alm:descriptor:prometheusEndpoint"},
	"status":             {"phase", "urn:alm:descriptor:io.kubernetes.phase"},
	"reason":             {"reason", "urn:alm:descriptor:io.kubernetes.phase:reason"},
}

var jsonTagRe = regexp.MustCompile("`json:\"([^,]+),?.*\"`")

func guessPathAndXDescriptorFromTag(tag string, isSpec bool) (path string, xd []string) {
	tagMatches := jsonTagRe.FindStringSubmatch(tag)
	if len(tagMatches) == 2 {
		path = tagMatches[1]
	}
	if isSpec {
		pathAndXD, ok := specXDescriptors[path]
		if ok {
			path, xd = pathAndXD[0], []string{pathAndXD[1]}
		}
	} else {
		pathAndXD, ok := statusXDescriptors[path]
		if ok {
			path, xd = pathAndXD[0], []string{pathAndXD[1]}
		}
	}
	return path, xd
}
