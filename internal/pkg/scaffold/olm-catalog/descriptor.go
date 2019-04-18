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
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/annotations"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/projutil"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/gengo/parser"
	"k8s.io/gengo/types"
)

const csvgenPrefix = annotations.SDKPrefix + ":gen-csv:"

// setCRDDescriptorForGVK parses type and struct field declaration comments on
// API types to populate a csv's spec.customresourcedefinitions.owned fields
// for a given API identified by Group, Version, and Kind.
func setCRDDescriptorForGVK(crdDesc *olmapiv1alpha1.CRDDescription, gvk schema.GroupVersionKind) error {
	group := gvk.Group
	if strings.Contains(group, ".") {
		group = strings.Split(gvk.Group, ".")[0]
	}
	apisDir := filepath.Join(scaffold.ApisDir, group, gvk.Version)
	if _, err := os.Stat(apisDir); err != nil && os.IsNotExist(err) {
		log.Infof(`API "%s" does not exist. Skipping CSV annotation parsing for this API.`, gvk)
		return nil
	}
	specType, statusType, pkgTypes, err := getSpecStatusPkgTypesForGVK(apisDir, gvk)
	if err != nil {
		return errors.Wrapf(err, `get spec, status, and package types for "%s"`, gvk)
	}

	var descriptors []descriptor
	for _, t := range pkgTypes {
		switch t.Kind {
		case types.Struct:
			if t.Name.Name == gvk.Kind {
				comments := append(t.SecondClosestCommentLines, t.CommentLines...)
				desc, err := parseCSVGenAnnotations(comments)
				if err != nil {
					return err
				}
				crdDesc.Description = parseDescription(comments)
				crdDesc.DisplayName = desc.displayName
				crdDesc.Resources = append(crdDesc.Resources, desc.resources...)
			}
			for _, m := range t.Members {
				desc, err := parseCSVGenAnnotations(m.CommentLines)
				if err != nil {
					return err
				}
				for _, d := range desc.descriptors {
					d.parentType = t
					d.member = m
					setDescriptorDefaultsIfEmpty(&d, m)
					descriptors = append(descriptors, d)
				}
			}
		}
	}
	crdDesc.Resources = sortResources(crdDesc.Resources)

	descriptors = mergeChildDescriptorPaths(specType, statusType, descriptors)
	// Now that we've merged child paths, ensure all possible x-descriptors
	// are added.
	for _, d := range descriptors {
		setXDescriptors(&d)
	}
	descriptors = sortDescriptors(descriptors)
	for _, d := range descriptors {
		switch d.descType {
		case typeSpec:
			crdDesc.SpecDescriptors = append(crdDesc.SpecDescriptors, olmapiv1alpha1.SpecDescriptor{
				Description:  d.description,
				DisplayName:  d.displayName,
				Path:         d.path,
				XDescriptors: d.xdesc,
			})
		case typeStatus:
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

func getSpecStatusPkgTypesForGVK(apisDir string, gvk schema.GroupVersionKind) (spec, status *types.Type, pkgTypes []*types.Type, err error) {
	p := parser.New()
	if err := p.AddDirRecursive("./" + apisDir); err != nil {
		return nil, nil, nil, err
	}
	universe, err := p.FindTypes()
	if err != nil {
		return nil, nil, nil, err
	}

	pp := strings.TrimSuffix(projutil.CheckAndGetProjectGoPkg(), apisDir)
	for _, pkg := range universe {
		if !strings.HasPrefix(pkg.Path, pp) && !strings.HasPrefix(pkg.Path, "./") {
			continue
		}
		for _, t := range pkg.Types {
			pkgTypes = append(pkgTypes, t)
			if t.Name.Name == gvk.Kind {
				for _, m := range t.Members {
					path := parsePathFromJSONTags(m.Tags)
					if path == "spec" {
						spec = m.Type
					} else if path == "status" {
						status = m.Type
					}
					if spec != nil && status != nil {
						break
					}
				}
			}
		}
	}
	if spec == nil {
		return nil, nil, nil, fmt.Errorf("no spec found in type %s", gvk.Kind)
	} else if status == nil {
		return nil, nil, nil, fmt.Errorf("no status found in type %s", gvk.Kind)
	}
	return spec, status, pkgTypes, nil
}

type descriptorType = string

const (
	typeSpec   descriptorType = "spec"
	typeStatus descriptorType = "status"
)

type descriptor struct {
	include     bool
	parentType  *types.Type
	member      types.Member
	descType    descriptorType
	description string
	displayName string
	path        string
	xdesc       []string
}

func sortDescriptors(ds []descriptor) []descriptor {
	sort.Slice(ds, func(i, j int) bool {
		return ds[i].displayName < ds[j].displayName
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

func wrapParseErr(err error) error {
	return errors.Wrap(err, "error parsing csv-gen annotation")
}

func parseCSVGenAnnotations(comments []string) (desc parsedCRDDescriptions, err error) {
	tags := types.ExtractCommentTags(csvgenPrefix, comments)
	spec, status := descriptor{descType: typeSpec}, descriptor{descType: typeStatus}
	for path, vals := range tags {
		pathElems, err := annotations.SplitPath(path)
		if err != nil {
			return desc, wrapParseErr(err)
		}
		parentPathElem, childPathElems := pathElems[0], pathElems[1:]
		switch parentPathElem {
		case "customresourcedefinitions":
			switch childPathElems[0] {
			case "specDescriptors":
				err = parseDescriptor(&spec, childPathElems, vals[0])
				if err != nil {
					return desc, wrapParseErr(err)
				}
			case "statusDescriptors":
				err = parseDescriptor(&status, childPathElems, vals[0])
				if err != nil {
					return desc, wrapParseErr(err)
				}
			case "displayName":
				desc.displayName, err = strconv.Unquote(vals[0])
				if err != nil {
					return desc, fmt.Errorf("error unquoting %s: %v", vals[0], err)
				}
			case "resources":
				for _, v := range vals {
					r, err := parseResource(v)
					if err != nil {
						return desc, fmt.Errorf("error parsing resource %s: %v", v, err)
					}
					desc.resources = append(desc.resources, r)
				}
			default:
				return desc, wrapParseErr(fmt.Errorf(`unsupported %s child path element "%s"`, parentPathElem, childPathElems[0]))
			}
		default:
			return desc, wrapParseErr(fmt.Errorf(`unsupported path element "%s"`, parentPathElem))
		}
	}

	for _, d := range []descriptor{spec, status} {
		if d.include {
			desc.descriptors = append(desc.descriptors, d)
		}
	}
	return desc, nil
}

func parseDescriptor(desc *descriptor, pathElems []string, val string) (err error) {
	switch len(pathElems) {
	case 1:
		desc.include, err = strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error parsing %s bool val '%s': %v", pathElems[0], val, err)
		}
	case 2:
		switch pathElems[1] {
		case "displayName":
			desc.displayName, err = strconv.Unquote(val)
			if err != nil {
				return fmt.Errorf("error unquoting %s: %v", val, err)
			}
		case "x-descriptors":
			xdStr, err := strconv.Unquote(val)
			if err != nil {
				return fmt.Errorf("error unquoting %s: %v", val, err)
			}
			desc.xdesc = strings.Split(xdStr, ",")
		default:
			return fmt.Errorf(`unsupported descriptor path element "%s"`, pathElems[1])
		}
	default:
		return fmt.Errorf(`unsupported descriptor path "%s"`, annotations.JoinPath(pathElems...))
	}
	return nil
}

func parseResource(rStr string) (r olmapiv1alpha1.APIResourceReference, err error) {
	rStr, err = strconv.Unquote(rStr)
	if err != nil {
		return r, err
	}
	rSplit := strings.SplitN(rStr, ",", 3)
	if len(rSplit) < 2 {
		return r, fmt.Errorf("resource string %s did not have at least a kind and a version", rStr)
	}
	r.Kind, r.Version = rSplit[0], rSplit[1]
	if len(rSplit) == 3 {
		r.Name, err = strconv.Unquote(rSplit[2])
		if err != nil {
			return r, err
		}
	}
	return r, nil
}

func setDescriptorDefaultsIfEmpty(desc *descriptor, m types.Member) {
	desc.description = parseDescription(m.CommentLines)
	desc.path = parsePathFromJSONTags(m.Tags)
	if desc.displayName == "" {
		desc.displayName = getDisplayName(m.Name)
	}
	setXDescriptors(desc)
}

func setXDescriptors(desc *descriptor) {
	switch desc.descType {
	case typeSpec:
		desc.xdesc = getSpecXDescriptorsByPath(desc.xdesc, desc.path)
	case typeStatus:
		desc.xdesc = getStatusXDescriptorsByPath(desc.xdesc, desc.path)
	}
}

func getTypeName(t *types.Type) string {
	nameSplit := strings.Split(t.Name.Name, ".")
	return nameSplit[len(nameSplit)-1]
}

func typeNamesEqual(t1, t2 *types.Type) bool {
	return getTypeName(t1) == getTypeName(t2)
}

func mergeChildDescriptorPaths(specType, statusType *types.Type, descriptors []descriptor) (newDescs []descriptor) {
	descMap := map[string][]descriptor{}
	for _, d := range descriptors {
		n := getTypeName(d.member.Type)
		descMap[n] = append(descMap[n], d)
	}
	bfsJoinDescriptorPaths(specType, typeSpec, descMap)
	bfsJoinDescriptorPaths(statusType, typeStatus, descMap)
	for _, ds := range descMap {
		for _, d := range ds {
			newDescs = append(newDescs, d)
		}
	}
	return newDescs
}

func bfsJoinDescriptorPaths(parentType *types.Type, pt descriptorType, descMap map[string][]descriptor) {
	nextMembers := parentType.Members
	level, lenNextMembers := 0, len(nextMembers)
	// BFS up to 5 levels.
	for len(nextMembers) > 0 && level < 5 {
		for _, m := range nextMembers {
			t := m.Type
			switch m.Type.Kind {
			case types.Map, types.Slice, types.Pointer, types.Chan:
				t = t.Elem
			case types.Alias, types.DeclarationOf:
				t = t.Underlying
			}
			if t.IsPrimitive() {
				continue
			}
			for _, mm := range t.Members {
				mn := getTypeName(mm.Type)
				if ds, ok := descMap[mn]; ok {
					for i := 0; i < len(ds); i++ {
						typesEqual := typeNamesEqual(m.Type, ds[i].parentType)
						membersEqual := mm.Name == ds[i].member.Name
						if ds[i].descType == pt && typesEqual && membersEqual {
							tags := parsePathFromJSONTags(m.Tags)
							if tags != "" && tags != typeSpec && tags != typeStatus {
								ds[i].path = tags + "." + ds[i].path
							}
						}
					}
					descMap[mn] = ds
				}
				nextMembers = append(nextMembers, mm)
			}
		}
		nextMembers = nextMembers[lenNextMembers:]
		lenNextMembers = len(nextMembers)
		level++
	}
}

// parseDescription joins comment strings into one line, removing any tool
// directives.
func parseDescription(comments []string) string {
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

func parsePathFromJSONTags(tags string) string {
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

// getSpecXDescriptorsByPath uses path's elements to get x-descriptors a CRD
// descriptor should have.
func getSpecXDescriptorsByPath(existingXDescs []string, path string) []string {
	return getXDescriptorsByPath(specXDescriptors, existingXDescs, path)
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
