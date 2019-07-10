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
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/operator-framework/operator-sdk/internal/annotations"
	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
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
// TODO(estroz): support ActionDescriptors parsing/setting.
func setCRDDescriptorForGVK(crdDesc *olmapiv1alpha1.CRDDescription, gvk schema.GroupVersionKind) error {
	if strings.Contains(gvk.Group, ".") {
		gvk.Group = strings.Split(gvk.Group, ".")[0]
	}
	apisDir := filepath.Join(scaffold.ApisDir, gvk.Group, gvk.Version)
	if _, err := os.Stat(apisDir); err != nil {
		if os.IsNotExist(err) {
			log.Infof(`API "%s" does not exist. Skipping CSV annotation parsing for this API.`, gvk)
			return nil
		}
		return err
	}
	p := parser.New()
	if err := p.AddDirRecursive("./" + apisDir); err != nil {
		return err
	}
	universe, err := p.FindTypes()
	if err != nil {
		return err
	}
	apiPkg := path.Join(projutil.GetGoPkg(), apisDir)
	specType, statusType, pkgTypes, err := getSpecStatusPkgTypesForAPI(universe, apiPkg, gvk.Kind)
	if err != nil {
		return errors.Wrapf(err, `get spec, status, and package types for "%s"`, gvk)
	}

	var descriptors []descriptor
	for _, t := range pkgTypes {
		switch t.Kind {
		case types.Struct:
			if t.Name.Name == gvk.Kind {
				comments := append(t.SecondClosestCommentLines, t.CommentLines...)
				pd, err := parseCSVGenAnnotations(comments)
				if err != nil {
					return err
				}
				crdDesc.Description = parseDescription(comments)
				crdDesc.DisplayName = pd.displayName
				crdDesc.Resources = append(crdDesc.Resources, pd.resources...)
			}
			for _, m := range t.Members {
				pd, err := parseCSVGenAnnotations(m.CommentLines)
				if err != nil {
					return err
				}
				for _, d := range pd.descriptors {
					d.parentType, d.member = t, m
					descriptors = append(descriptors, d)
				}
			}
		}
	}

	crdDesc.Resources = sortResources(crdDesc.Resources)
	descriptors = mergeChildDescriptorPaths(specType, statusType, descriptors)
	// Now that we've merged child paths, ensure all fields not set are added.
	for i := 0; i < len(descriptors); i++ {
		setDescriptorDefaultsIfEmpty(&descriptors[i])
	}
	for _, d := range sortDescriptors(descriptors) {
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

// getSpecStatusPkgTypesForAPI finds and returns types {kind}Spec, {kind}Status,
// and all types in apiPkg.
func getSpecStatusPkgTypesForAPI(universe types.Universe, apiPkg, kind string) (spec, status *types.Type, pkgTypes []*types.Type, err error) {
	for _, pkg := range universe {
		if pkg.Path != apiPkg && !strings.HasPrefix(pkg.Path, "./") {
			continue
		}
		for _, t := range pkg.Types {
			pkgTypes = append(pkgTypes, t)
			if t.Name.Name == kind {
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
		return nil, nil, nil, fmt.Errorf("no spec found in type %s", kind)
	}
	if status == nil {
		return nil, nil, nil, fmt.Errorf("no status found in type %s", kind)
	}
	if len(pkgTypes) == 0 {
		return nil, nil, nil, fmt.Errorf("no package types found in API %s", apiPkg)
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
		return ds[i].path < ds[j].path
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
			return pd, wrapParseErr(err)
		}
		parentPathElem, childPathElems := pathElems[0], pathElems[1:]
		switch parentPathElem {
		case "customresourcedefinitions":
			switch childPathElems[0] {
			case "specDescriptors":
				err = parseDescriptor(&specd, childPathElems, vals[0])
				if err != nil {
					return pd, wrapParseErr(err)
				}
			case "statusDescriptors":
				err = parseDescriptor(&statusd, childPathElems, vals[0])
				if err != nil {
					return pd, wrapParseErr(err)
				}
			case "displayName":
				pd.displayName, err = strconv.Unquote(vals[0])
				if err != nil {
					return pd, fmt.Errorf("error unquoting %s: %v", vals[0], err)
				}
			case "resources":
				for _, v := range vals {
					r, err := parseResource(v)
					if err != nil {
						return pd, fmt.Errorf("error parsing resource %s: %v", v, err)
					}
					pd.resources = append(pd.resources, r)
				}
			default:
				return pd, wrapParseErr(fmt.Errorf(`unsupported %s child path element "%s"`, parentPathElem, childPathElems[0]))
			}
		default:
			return pd, wrapParseErr(fmt.Errorf(`unsupported path element "%s"`, parentPathElem))
		}
	}

	for _, d := range []descriptor{specd, statusd} {
		if d.include {
			pd.descriptors = append(pd.descriptors, d)
		}
	}
	return pd, nil
}

// parseDescriptor determines which descriptor annotation was passed from
// pathElems and sets val to the corresponding field in d.
func parseDescriptor(d *descriptor, pathElems []string, val string) (err error) {
	switch len(pathElems) {
	case 1:
		d.include, err = strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("error parsing %s bool val '%s': %v", pathElems[0], val, err)
		}
	case 2:
		switch pathElems[1] {
		case "displayName":
			d.displayName, err = strconv.Unquote(val)
			if err != nil {
				return fmt.Errorf("error unquoting %s: %v", val, err)
			}
		case "x-descriptors":
			xdStr, err := strconv.Unquote(val)
			if err != nil {
				return fmt.Errorf("error unquoting %s: %v", val, err)
			}
			d.xdesc = strings.Split(xdStr, ",")
		default:
			return fmt.Errorf(`unsupported descriptor path element "%s"`, pathElems[1])
		}
	default:
		return fmt.Errorf(`unsupported descriptor path "%s"`, annotations.JoinPath(pathElems...))
	}
	return nil
}

// parseResource parses a resource string of the form:
// "kind,version,\"quoted name\""
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

// setDescriptorDefaultsIfEmpty sets d's fields by parsing values from their
// typical locations in data contained in d, ex. d.member, but only if those
// fields are empty or should be overwritten.
func setDescriptorDefaultsIfEmpty(d *descriptor) {
	if d.description == "" {
		d.description = parseDescription(d.member.CommentLines)
	}
	if d.path == "" {
		d.path = parsePathFromJSONTags(d.member.Tags)
	}
	if d.displayName == "" {
		d.displayName = k8sutil.GetDisplayName(d.member.Name)
	}
	switch d.descType {
	case typeSpec:
		d.xdesc = getSpecXDescriptorsByPath(d.xdesc, d.path)
	case typeStatus:
		d.xdesc = getStatusXDescriptorsByPath(d.xdesc, d.path)
	}
}

// mergeChildDescriptorPaths joins all child descriptor paths with their
// parents, and returns the updated descriptors.
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

type memberNode struct {
	types.Member
	parentNode *memberNode
}

type descNodeMapping struct {
	parentNode *memberNode
	descriptor descriptor
}

// bfsJoinDescriptorPaths performs BFS on all struct members in parentType to
// find members corresponding to descriptors in descMap, which contain the
// member they were parsed from. pt determines parentType;s descriptor type,
// ex. "spec", "status".
func bfsJoinDescriptorPaths(parentType *types.Type, pt descriptorType, descMap map[string][]descriptor) {
	nextMembers, leaves := []*memberNode{}, []descNodeMapping{}
	for _, m := range parentType.Members {
		nextMembers = append(nextMembers, &memberNode{m, nil})
	}
	maxLevel := 10
	level, lenNextMembers := 0, len(nextMembers)
	// BFS up to maxLevel for qualifying leaves. We must check that both the
	// parent type and member type, and member names are equal before adding a
	// leaf. We must loop through all fields in a parent struct type, not just
	// all members in nextMembers, in order to do so. We could find an incorrect
	// leaf if we only checked member type/name equality since different structs
	// can have the same field signatures.
	for len(nextMembers) > 0 && level < maxLevel {
		for _, m := range nextMembers {
			t := getUnderlyingType(m.Type)
			for _, mm := range t.Members {
				node := memberNode{mm, m}
				nextMembers = append(nextMembers, &node)
				mmn := getTypeName(mm.Type)
				if ds, ok := descMap[mmn]; ok {
					newDs := []descriptor{}
					for _, d := range ds {
						typesEqual := typeNamesEqual(m.Type, d.parentType)
						membersEqual := mm.Name == d.member.Name
						if d.descType == pt && typesEqual && membersEqual {
							leaves = append(leaves, descNodeMapping{&node, d})
						} else {
							newDs = append(newDs, d)
						}
					}
					descMap[mmn] = newDs
				}
			}
		}
		nextMembers = nextMembers[lenNextMembers:]
		lenNextMembers = len(nextMembers)
		level++
	}

	seenSpec, seenStatus := false, false
	for _, l := range leaves {
		segments := []string{}
		if l.descriptor.path != "" {
			segments = append(segments, l.descriptor.path)
		}
		for parent := l.parentNode; parent != nil; parent = parent.parentNode {
			pathSeg := ""
			// Use the field's name if it doesn't have a JSON tag.
			if parent.Tags == "" {
				pathSeg = parent.Name
			} else {
				pathSeg = parsePathFromJSONTags(parent.Tags)
			}
			if pathSeg != "" {
				// {Kind}Spec and {Kind}Status pathSegs should not be in the resulting
				// path, as spec/status is implied by specDescriptors/statusDescriptors;
				// children of these types with "spec"/"status" pathSegs should be
				// included.
				if pathSeg == typeSpec && !seenSpec {
					seenSpec = true
					continue
				}
				if pathSeg == typeStatus && !seenStatus {
					seenStatus = true
					continue
				}
				segments = append([]string{pathSeg}, segments...)
			}
		}
		l.descriptor.path = strings.Join(segments, ".")
		n := getTypeName(l.descriptor.member.Type)
		descMap[n] = append(descMap[n], l.descriptor)
	}
}

func getUnderlyingType(t *types.Type) *types.Type {
	switch t.Kind {
	case types.Map, types.Slice, types.Pointer, types.Chan:
		t = t.Elem
	case types.Alias, types.DeclarationOf:
		t = t.Underlying
	}
	return t
}

func getTypeName(t *types.Type) string {
	return getUnderlyingType(t).Name.String()
}

func typeNamesEqual(t1, t2 *types.Type) bool {
	return getTypeName(t1) == getTypeName(t2)
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
