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
	"sort"
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

// setCRDDescriptorsForGVK parses document and type declaration comments on
// CRD types to populate a csv's 'crds.owned[].{spec,status}Descriptors' for
// a given Group, Version, and Kind.
func setCRDDescriptorsForGVK(crdDesc *olmapiv1alpha1.CRDDescription, gvk schema.GroupVersionKind) error {
	projutil.MustInProjectRoot()

	group := gvk.Group
	if strings.Contains(group, ".") {
		group = strings.Split(gvk.Group, ".")[0]
	}
	apisDir := filepath.Join(scaffold.ApisDir, group, gvk.Version)
	p := parser.New()
	if err := p.AddDirRecursive("./" + apisDir); err != nil {
		return err
	}
	universe, err := p.FindTypes()
	if err != nil {
		return err
	}
	pp := strings.TrimSuffix(projutil.CheckAndGetProjectGoPkg(), apisDir)

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
					path := getPathFromJSONTags(m.Tags)
					if path == "spec" {
						specType = m.Type
					} else if path == "status" {
						statusType = m.Type
					}
					if specType != nil && statusType != nil {
						break
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
	// fmt.Println("kind:", gvk.Kind)

	var descriptors []descriptor
	for _, t := range pkgTypes {
		// fmt.Printf("\ntype %s\n", t.Name.Name)
		for _, m := range t.Members {
			// fmt.Printf("member %s %s\n", m.Name, m.Type.Name.Name)
			comments := m.CommentLines
			desc := processDescription(comments)
			// fmt.Printf("\tcomment lines: %+q\n\n", m.CommentLines)
			for len(comments) > 0 {
				d, cs, err := parseCSVGenAnnotation(m, comments)
				if err != nil {
					return err
				}
				d.description = desc
				descriptors = append(descriptors, d)
				comments = cs
			}
		}
	}
	sort.Slice(descriptors, func(i int, j int) bool {
		return descriptors[i].displayName < descriptors[j].displayName
	})

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

func parseCSVGenAnnotation(m types.Member, comments []string) (d descriptor, cs []string, err error) {
	if len(comments) == 0 {
		return descriptor{}, nil, nil
	}
	var numLinesParsed int
	var doneForKind bool
	for _, line := range comments {
		numLinesParsed++
		line = strings.TrimSpace(line)
		trimmed := strings.TrimPrefix(line, csvgenPrefix)
		if trimmed == line {
			continue
		}
		// fmt.Printf("parsing \"%s\"\n", line)
		keyValue := strings.Split(trimmed, "=")
		if len(keyValue) != 2 {
			return descriptor{}, nil, fmt.Errorf(`invalid descriptor format "%s"`, trimmed)
		}
		// fmt.Printf("keyValue \"%+q\"\n", keyValue)
		keyParts := strings.Split(keyValue[0], ".")
		if len(keyParts) == 0 {
			return descriptor{}, nil, fmt.Errorf(`invalid descriptor format "%s"`, keyValue[0])
		}
		// fmt.Printf("keyParts \"%+q\"\n", keyParts)
		val, keyType, keyPath := keyValue[1], keyParts[0], keyParts[1:]
		switch keyType {
		case "customresourcedefinitions":
			switch keyPath[0] {
			case "descriptor":
				switch len(keyPath) {
				case 1:
					p, err := strconv.Unquote(val)
					if err != nil {
						return descriptor{}, nil, fmt.Errorf("error unquoting %s: %v", val, err)
					}
					if p != "spec" && p != "status" {
						return descriptor{}, nil, fmt.Errorf("error parsing %s type %s: must be either spec or status", keyType, p)
					}
					d.spec = p == "spec"
					d.status = !d.spec
				case 2:
					switch keyPath[1] {
					case "displayName":
						d.displayName, err = strconv.Unquote(val)
						if err != nil {
							return descriptor{}, nil, fmt.Errorf("error unquoting %s: %v", val, err)
						}
					case "path":
						d.path, err = strconv.Unquote(val)
						if err != nil {
							return descriptor{}, nil, fmt.Errorf("error unquoting %s: %v", val, err)
						}
					case "x-descriptors":
						xdStr, err := strconv.Unquote(val)
						if err != nil {
							return descriptor{}, nil, fmt.Errorf("error unquoting %s: %v", val, err)
						}
						d.xdesc = strings.Split(xdStr, ",")
					}
				}
			case "kind":
				// Once we hit another "kind" descriptor, we've finished this block.
				if doneForKind {
					numLinesParsed--
					goto finishParse
				}
				d.kind, err = strconv.Unquote(val)
				if err != nil {
					return descriptor{}, nil, fmt.Errorf("error unquoting %s: %v", val, err)
				}
				doneForKind = true
			default:
				return descriptor{}, nil, fmt.Errorf(`error parsing csv-gen annotation: unsupported annotation "%s"`, keyType)
			}
		}
	}

finishParse:

	d.typ = m.Type
	if d.displayName == "" {
		d.displayName = getDisplayName(m.Name)
	}
	if d.path == "" {
		d.path = getPathFromJSONTags(m.Tags)
	}
	if len(d.xdesc) == 0 {
		d.xdesc = getXDescriptorByPath(d.path, d.spec)
	}
	if len(comments) > numLinesParsed {
		return d, comments[numLinesParsed:], nil
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

var jsonTagRe = regexp.MustCompile(`json:"([a-zA-Z0-9,]+)"`)

func getPathFromJSONTags(tags string) string {
	tagMatches := jsonTagRe.FindStringSubmatch(tags)
	if len(tagMatches) > 1 {
		ts := strings.Split(tagMatches[1], ",")
		if len(ts) != 0 && ts[0] != "" {
			return ts[0]
		}
	}
	return ""
}

// From https://github.com/openshift/console/blob/master/frontend/public/components/operator-lifecycle-manager/descriptors/types.ts#L5-L14
var specXDescriptors = map[string]string{
	"size":                 "urn:alm:descriptor:com.tectonic.ui:podCount",
	"podCount":             "urn:alm:descriptor:com.tectonic.ui:podCount",
	"endpoints":            "urn:alm:descriptor:com.tectonic.ui:endpointList",
	"endpointList":         "urn:alm:descriptor:com.tectonic.ui:endpointList",
	"label":                "urn:alm:descriptor:com.tectonic.ui:label",
	"resources":            "urn:alm:descriptor:com.tectonic.ui:resourceRequirements",
	"resourceRequirements": "urn:alm:descriptor:com.tectonic.ui:resourceRequirements",
	"selector":             "urn:alm:descriptor:com.tectonic.ui:selector:",
	"namespaceSelector":    "urn:alm:descriptor:com.tectonic.ui:namespaceSelector",
	"booleanSwitch":        "urn:alm:descriptor:com.tectonic.ui:booleanSwitch",
}

// From https://github.com/openshift/console/blob/master/frontend/public/components/operator-lifecycle-manager/descriptors/types.ts#L16-L27
var statusXDescriptors = map[string]string{
	"podStatuses":        "urn:alm:descriptor:com.tectonic.ui:podStatuses",
	"size":               "urn:alm:descriptor:com.tectonic.ui:podCount",
	"podCount":           "urn:alm:descriptor:com.tectonic.ui:podCount",
	"link":               "urn:alm:descriptor:org.w3:link",
	"w3link":             "urn:alm:descriptor:org.w3:link",
	"conditions":         "urn:alm:descriptor:io.kubernetes.conditions",
	"text":               "urn:alm:descriptor:text",
	"prometheusEndpoint": "urn:alm:descriptor:prometheusEndpoint",
	"phase":              "urn:alm:descriptor:io.kubernetes.phase",
	"k8sPhase":           "urn:alm:descriptor:io.kubernetes.phase",
	"reason":             "urn:alm:descriptor:io.kubernetes.phase:reason",
	"k8sReason":          "urn:alm:descriptor:io.kubernetes.phase:reason",
}

// getXDescriptorByPath uses a path name to get a likely x-descriptor a CRD
// descriptor should have.
func getXDescriptorByPath(path string, isSpec bool) []string {
	pathSplit := strings.Split(path, ".")
	tag := pathSplit[len(pathSplit)-1]
	if isSpec {
		xd, ok := specXDescriptors[tag]
		if ok {
			return []string{xd}
		}
	} else {
		xd, ok := statusXDescriptors[tag]
		if ok {
			return []string{xd}
		}
	}
	return nil
}
