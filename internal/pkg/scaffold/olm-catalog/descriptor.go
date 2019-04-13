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

package catalog

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/gengo/parser"
	"k8s.io/gengo/types"
)

const csvgenPrefix = "+operator-sdk:csv-gen:"

// setCRDDescriptorsForGVK2 parses document and type declaration comments on
// CRD types to populate a csv's 'crds.owned[].{spec,status}Descriptors' for
// a given Group, Version, and Kind.
func setCRDDescriptorsForGVK2(crdDesc *olmapiv1alpha1.CRDDescription, gvk schema.GroupVersionKind) error {
	projutil.MustInProjectRoot()

	group := gvk.Group
	if strings.Contains(group, ".") {
		group = strings.Split(gvk.Group, ".")[0]
	}
	dir := filepath.Join(scaffold.ApisDir, group, gvk.Version)
	p := parser.New()
	if err := p.AddDirRecursive("./" + dir); err != nil {
		return err
	}
	universe, err := p.FindTypes()
	if err != nil {
		return err
	}
	pp := projutil.CheckAndGetProjectGoPkg()

	var pkgTypes []*types.Type
	var specType, statusType *types.Type
	for _, pkg := range universe {
		if !strings.HasPrefix(pkg.Path, pp) {
			continue
		}
		for _, t := range pkg.Types {
			pkgTypes = append(pkgTypes, t)
			if t.Name.Name == gvk.Kind {
				for _, m := range t.Members {
					if m.Tags == "" {
						continue
					}
					tagMatches := jsonTagRe.FindStringSubmatch(m.Tags)
					if len(tagMatches) == 0 {
						continue
					}
					ts := strings.Split(tagMatches[1], ",")
					if len(ts) != 0 && ts[0] != "" {
						if ts[0] == "spec" {
							specType = m.Type
						} else if ts[0] == "status" {
							statusType = m.Type
						}
					}
				}
			}
		}
	}
	if specType.Name.Name == "" {
		return fmt.Errorf("no spec found in type %s", gvk.Kind)
	} else if statusType.Name.Name == "" {
		return fmt.Errorf("no status found in type %s", gvk.Kind)
	}
	fmt.Println("kind:", gvk.Kind)

	var descriptors []descriptor
	for _, t := range pkgTypes {
		// fmt.Printf("type %s\n", t.Name.Name)
		for _, m := range t.Members {
			// fmt.Printf("\tmember %s %s\n", m.Name, m.Type.Name.Name)
			comments := m.CommentLines
			desc := processDescription(comments)
			// fmt.Printf("\tcomment lines: %+q\n\n", m.CommentLines)
			for len(comments) > 0 {
				d, cs, err := parseCSVGenAnnotation(m, comments)
				if err != nil {
					return err
				}
				// fmt.Println("done parsing")
				d.description = desc
				descriptors = append(descriptors, d)
				comments = cs
			}
		}
	}

	for _, d := range descriptors {
		if d.spec && crdDesc.Kind == d.kind {
			crdDesc.SpecDescriptors = append(crdDesc.SpecDescriptors, olmapiv1alpha1.SpecDescriptor{
				Description:  d.description,
				DisplayName:  d.displayName,
				Path:         d.path,
				XDescriptors: d.xdesc,
			})
		} else if d.status && crdDesc.Kind == d.kind {
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

type descriptor struct {
	typ          *types.Type
	kind         string
	spec, status bool

	description string
	displayName string
	path        string
	xdesc       []string
}

var jsonTagRe = regexp.MustCompile(`json:"([a-zA-Z0-9,]+)"`)

func parseCSVGenAnnotation(m types.Member, comments []string) (d descriptor, cs []string, err error) {
	d.typ = m.Type
	var numLinesParsed int
	var doneForKind bool
	for _, line := range comments {
		line = strings.TrimSpace(line)
		trimmed := strings.TrimPrefix(line, csvgenPrefix)
		if trimmed == line {
			continue
		}
		// fmt.Printf("parsing \"%s\"\n", line)
		parts := strings.Split(trimmed, "=")
		if len(parts) != 2 {
			return descriptor{}, nil, fmt.Errorf(`invalid descriptor format "%s"`, trimmed)
		}
		// fmt.Printf("parts \"%+q\"\n", parts)
		aSplit := strings.Split(parts[0], ".")
		if len(aSplit) == 0 {
			return descriptor{}, nil, fmt.Errorf(`invalid descriptor format "%s"`, parts[0])
		}
		// fmt.Printf("aSplit \"%+q\"\n", aSplit)
		numLinesParsed++
		switch aSplit[0] {
		case "customresourcedefinitions":
			switch aSplit[1] {
			case "descriptor":
				p, err := strconv.Unquote(parts[1])
				if err != nil {
					return descriptor{}, nil, fmt.Errorf("error unquoting %s: %v", parts[1], err)
				}
				if p != "spec" && p != "status" {
					return descriptor{}, nil, fmt.Errorf("error parsing %s type %s: must be either spec or status", aSplit[0], p)
				}
				d.spec = p == "spec"
				d.status = !d.spec
			case "kind":
				// Once we hit another "kind" descriptor, we've finished this block.
				if doneForKind {
					break
				}
				d.kind, err = strconv.Unquote(parts[1])
				if err != nil {
					return descriptor{}, nil, fmt.Errorf("error unquoting %s: %v", parts[1], err)
				}
				doneForKind = true
			case "displayName":
				d.displayName, err = strconv.Unquote(parts[1])
				if err != nil {
					return descriptor{}, nil, fmt.Errorf("error unquoting %s: %v", parts[1], err)
				}
			case "path":
				d.path, err = strconv.Unquote(parts[1])
				if err != nil {
					return descriptor{}, nil, fmt.Errorf("error unquoting %s: %v", parts[1], err)
				}
			case "x-descriptors":
				xdStr, err := strconv.Unquote(parts[1])
				if err != nil {
					return descriptor{}, nil, fmt.Errorf("error unquoting %s: %v", parts[1], err)
				}
				d.xdesc = strings.Split(xdStr, ",")
			default:
				numLinesParsed--
			}
		}
	}
	if d.displayName == "" {
		d.displayName = getDisplayName(m.Name)
	}
	if d.path == "" {
		tagMatches := jsonTagRe.FindStringSubmatch(m.Tags)
		if len(tagMatches) != 0 {
			ts := strings.Split(tagMatches[1], ",")
			if len(ts) != 0 && ts[0] != "" {
				d.path = ts[0]
			} else {
				d.path = strings.ToLower(string(m.Name[0])) + m.Name[1:]
			}
		}
	}
	if len(d.xdesc) == 0 {
		d.xdesc = getXDescriptorByPath(d.path, d.spec)
	}
	return d, nil, nil
}

// processDescription joins comment strings into one line, removing any tool
// directives.
func processDescription(comments []string) string {
	var lines []string
	for _, c := range comments {
		l := strings.TrimSpace(strings.TrimLeft(c, "/"))
		if l == "" || strings.Contains(l, "+") {
			continue
		}
		lines = append(lines, l)
	}
	return strings.Join(lines, " ")
}

// From https://github.com/openshift/console/blob/master/frontend/public/components/operator-lifecycle-manager/descriptors/types.ts#L5-L14
var specXDescriptors = map[string][]string{
	"size":              []string{"urn:alm:descriptor:com.tectonic.ui:podCount"},
	"endpoints":         []string{"urn:alm:descriptor:com.tectonic.ui:endpointList"},
	"label":             []string{"urn:alm:descriptor:com.tectonic.ui:label"},
	"resources":         []string{"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"},
	"selector":          []string{"urn:alm:descriptor:com.tectonic.ui:selector:"},
	"namespaceSelector": []string{"urn:alm:descriptor:com.tectonic.ui:namespaceSelector"},
	"booleanSwitch":     []string{"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"},
}

// From https://github.com/openshift/console/blob/master/frontend/public/components/operator-lifecycle-manager/descriptors/types.ts#L16-L27
var statusXDescriptors = map[string][]string{
	"size":               []string{"urn:alm:descriptor:com.tectonic.ui:podCount"},
	"podStatuses":        []string{"urn:alm:descriptor:com.tectonic.ui:podStatuses"},
	"links":              []string{"urn:alm:descriptor:org.w3:link"},
	"conditions":         []string{"urn:alm:descriptor:io.kubernetes.conditions"},
	"text":               []string{"urn:alm:descriptor:text"},
	"prometheusEndpoint": []string{"urn:alm:descriptor:prometheusEndpoint"},
	"status":             []string{"urn:alm:descriptor:io.kubernetes.phase"},
	"reason":             []string{"urn:alm:descriptor:io.kubernetes.phase:reason"},
}

// getXDescriptorByPath uses a path name to get a likely x-descriptor a CRD
// descriptor should have.
func getXDescriptorByPath(path string, isSpec bool) (xd []string) {
	pathSplit := strings.Split(path, ".")
	tag := pathSplit[len(pathSplit)-1]
	if isSpec {
		return specXDescriptors[tag]
	}
	return statusXDescriptors[tag]
}
