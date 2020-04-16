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
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/operator-framework/operator-sdk/internal/annotations"

	"github.com/fatih/structtag"
	olmapiv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/gengo/types"
)

const csvgenPrefix = annotations.SDKPrefix + ":gen-csv:"

type descriptorType = string

const (
	typeSpec   descriptorType = "spec"
	typeStatus descriptorType = "status"
)

type descriptor struct {
	// Use a SpecDescriptor since it has the same fields as a StatusDescriptor.
	olmapiv1alpha1.SpecDescriptor
	include  bool
	descType descriptorType
}

func sortDescriptors(ds []descriptor) []descriptor {
	sort.Slice(ds, func(i, j int) bool {
		return ds[i].Path < ds[j].Path
	})
	return ds
}

type parsedCRDDescriptions struct {
	descriptors []descriptor
	displayName string
	resources   []olmapiv1alpha1.APIResourceReference
}

func sortResources(rs []olmapiv1alpha1.APIResourceReference) []olmapiv1alpha1.APIResourceReference {
	sort.Slice(rs, func(i, j int) bool {
		return rs[i].Kind < rs[j].Kind
	})
	return rs
}

// parseCSVGenAnnotations parses all descriptor annotations from comments,
// each of which should contain one spec.customresourcedefinitions.owned entry.
// field Once all comments have been parsed, the entry is added to a
// parsedCRDDescriptions.
func parseCSVGenAnnotations(comments []string) (pd parsedCRDDescriptions, err error) {
	tags := types.ExtractCommentTags(csvgenPrefix, comments)
	specd, statusd := descriptor{descType: typeSpec}, descriptor{descType: typeStatus}
	for path, vals := range tags {
		pathElems, err := annotations.SplitPath(path)
		if err != nil {
			return parsedCRDDescriptions{}, err
		}
		parentPathElem, childPathElems := pathElems[0], pathElems[1:]
		switch parentPathElem {
		case "customresourcedefinitions":
			switch childPathElems[0] {
			case "specDescriptors":
				err = parseMemberAnnotation(&specd, childPathElems, vals[0])
				if err != nil {
					return parsedCRDDescriptions{}, err
				}
			case "statusDescriptors":
				err = parseMemberAnnotation(&statusd, childPathElems, vals[0])
				if err != nil {
					return parsedCRDDescriptions{}, err
				}
			case "displayName":
				pd.displayName, err = strconv.Unquote(vals[0])
				if err != nil {
					return parsedCRDDescriptions{}, fmt.Errorf("error unquoting displayName %s: %v", vals[0], err)
				}
			case "resources":
				for _, v := range vals {
					r, err := parseResource(v)
					if err != nil {
						return parsedCRDDescriptions{}, fmt.Errorf("error parsing resource %s: %v", v, err)
					}
					pd.resources = append(pd.resources, r)
				}
			default:
				return parsedCRDDescriptions{}, fmt.Errorf("unsupported %s child path element %s",
					parentPathElem, childPathElems[0])
			}
		default:
			return parsedCRDDescriptions{}, fmt.Errorf("unsupported path element %s", parentPathElem)
		}
	}
	pd.descriptors = append(pd.descriptors, specd, statusd)
	return pd, nil
}

// parseMemberAnnotation determines which descriptor annotation was passed from
// pathElems and sets val to the corresponding field in d.
func parseMemberAnnotation(d *descriptor, pathElems []string, val string) (err error) {
	switch len(pathElems) {
	case 1:
		// If this case is never entered, d will not be included.
		d.include, err = strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error parsing %s bool val %s: %v", pathElems[0], val, err)
		}
	case 2:
		switch pathElems[1] {
		case "displayName":
			d.DisplayName, err = strconv.Unquote(val)
			if err != nil {
				return fmt.Errorf("error unquoting field displayName %s: %v", val, err)
			}
		case "x-descriptors":
			xdStr, err := strconv.Unquote(val)
			if err != nil {
				return fmt.Errorf("error unquoting field x-descriptors %s: %v", val, err)
			}
			d.XDescriptors = strings.Split(xdStr, ",")
		default:
			return fmt.Errorf("unsupported descriptor path element %s", pathElems[1])
		}
	default:
		return fmt.Errorf("unsupported descriptor path %s", annotations.JoinPath(pathElems...))
	}
	return nil
}

// parseResource parses a resource string of the form:
// "kind,version,\"quoted name\""
func parseResource(rStr string) (r olmapiv1alpha1.APIResourceReference, err error) {
	rStr, err = strconv.Unquote(rStr)
	if err != nil {
		return r, fmt.Errorf("error unquoting resource %s: %v", rStr, err)
	}
	rSplit := strings.SplitN(rStr, ",", 3)
	if len(rSplit) < 2 {
		return r, fmt.Errorf("resource string %s did not have at least a kind and a version", rStr)
	}
	r.Kind, r.Version = strings.TrimSpace(rSplit[0]), strings.TrimSpace(rSplit[1])
	if len(rSplit) == 3 {
		r.Name, err = strconv.Unquote(rSplit[2])
		if err != nil {
			return r, fmt.Errorf("error unquoting resource name %s: %v", rSplit[2], err)
		}
		r.Name = strings.TrimSpace(r.Name)
	}
	return r, nil
}

// parseDescription joins comment strings into one line, removing any tool
// directives.
func parseDescription(comments []string) string {
	var lines []string
	for _, c := range comments {
		l := strings.TrimSpace(strings.TrimLeft(c, "/"))
		if l == "" || strings.HasPrefix(l, "+") {
			continue
		}
		lines = append(lines, l)
	}
	return strings.Join(lines, " ")
}

const (
	inlinedTag = "##inline##"
	ignoredTag = "##ignore##"
)

// getPathFromMember constructs a path from data in member, either from
// its struct tags or name.
func getPathFromMember(member types.Member) (string, error) {
	// Embedded fields are inlined and children may be included.
	if member.Embedded {
		return inlinedTag, nil
	}
	// Unexported fields should be ignored in downstream processing.
	if isNotExported(member.Name) {
		return ignoredTag, nil
	}
	tags, err := structtag.Parse(member.Tags)
	if err != nil {
		return "", err
	}
	jsonTag, err := tags.Get("json")
	if err == nil {
		// Parse returns an error if no JSON tag is in tags, at which point we'll
		// use another method to get path.
		switch {
		case contains(jsonTag.Options, "inline"):
			return inlinedTag, nil
		case jsonTag.Name == "-":
			if len(jsonTag.Options) == 0 {
				return ignoredTag, nil
			}
			return jsonTag.Name, nil
		case jsonTag.Name != "":
			return jsonTag.Name, nil
		}
	}
	// There is no JSON tag in tags or tag name is empty.
	// Use member name as path as json.Marshal does.
	return member.Name, nil
}

// isNotExported returns true if name is not an exported struct field name.
func isNotExported(name string) bool {
	if len(name) == 0 {
		return true
	}
	return unicode.IsLower(rune(name[0]))
}

func contains(options []string, key string) bool {
	for _, opt := range options {
		if opt == key {
			return true
		}
	}
	return false
}

func isPathInline(path string) bool {
	return path == inlinedTag
}

func isPathIgnore(path string) bool {
	return path == ignoredTag
}

//nolint:lll
// From https://github.com/openshift/console/blob/feabd61/frontend/packages/operator-lifecycle-manager/src/components/descriptors/types.ts#L3-L26
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

	"password":        "urn:alm:descriptor:com.tectonic.ui:password",
	"checkbox":        "urn:alm:descriptor:com.tectonic.ui:checkbox",
	"imagePullPolicy": "urn:alm:descriptor:com.tectonic.ui:imagePullPolicy",
	"updateStrategy":  "urn:alm:descriptor:com.tectonic.ui:updateStrategy",
	"text":            "urn:alm:descriptor:com.tectonic.ui:text",
	"number":          "urn:alm:descriptor:com.tectonic.ui:number",
	"nodeAffinity":    "urn:alm:descriptor:com.tectonic.ui:nodeAffinity",
	"podAffinity":     "urn:alm:descriptor:com.tectonic.ui:podAffinity",
	"podAntiAffinity": "urn:alm:descriptor:com.tectonic.ui:podAntiAffinity",
	"advanced":        "urn:alm:descriptor:com.tectonic.ui:advanced",
}

// getSpecXDescriptorsByPath uses path's elements to get x-descriptors a CRD
// descriptor should have.
func getSpecXDescriptorsByPath(existingXDescs []string, path string) []string {
	return getXDescriptorsByPath(specXDescriptors, existingXDescs, path)
}

//nolint:lll // the  following line is  a reference, if split it will not be clear if the whole line is a link
// From https://github.com/openshift/console/blob/feabd61/frontend/packages/operator-lifecycle-manager/src/components/descriptors/types.ts#L28-L39
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

// getStatusXDescriptorsByPath uses path's elements to get x-descriptors a CRD
// descriptor should have.
func getStatusXDescriptorsByPath(existingXDescs []string, path string) []string {
	return getXDescriptorsByPath(statusXDescriptors, existingXDescs, path)
}

func getXDescriptorsByPath(relevantXDescs map[string]string, existingXDescs []string, path string) (xdescs []string) {
	// Ensure no duplicate x-descriptors are returned.
	xdescMap := map[string]struct{}{}
	for _, xd := range existingXDescs {
		xdescMap[xd] = struct{}{}
	}
	pathSplit := strings.Split(path, ".")
	for _, tag := range pathSplit {
		xd, ok := relevantXDescs[tag]
		if ok {
			xdescMap[xd] = struct{}{}
		}
	}
	for xd := range xdescMap {
		xdescs = append(xdescs, xd)
	}
	sort.Strings(xdescs)
	return xdescs
}
