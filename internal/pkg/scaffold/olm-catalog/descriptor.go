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
	"github.com/pkg/errors"

	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/gengo/parser"
	"k8s.io/gengo/types"
)

const csvgenPrefix = annotations.SDKPrefix + ":csv-gen:"

// setCRDDescriptorsForGVK parses document and type declaration comments on
// CRD types to populate a csv's 'crds.owned[].{spec,status}Descriptors' for
// a given Group, Version, and Kind.
func setCRDDescriptorsForGVK(crdDesc *olmapiv1alpha1.CRDDescription, gvk schema.GroupVersionKind) error {
	group := gvk.Group
	if strings.Contains(group, ".") {
		group = strings.Split(gvk.Group, ".")[0]
	}
	apisDir := filepath.Join(scaffold.ApisDir, group, gvk.Version)
	if _, err := os.Stat(apisDir); err != nil && os.IsNotExist(err) {
		log.Infof(`API "%s" does not exist. Skipping CSV annotation parsing for this API.`, gvk)
		return nil
	}
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
		if !strings.HasPrefix(pkg.Path, pp) && !strings.HasPrefix(pkg.Path, "./") {
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
	if specType == nil {
		return fmt.Errorf("no spec found in type %s", gvk.Kind)
	} else if statusType == nil {
		return fmt.Errorf("no status found in type %s", gvk.Kind)
	}
	// fmt.Println("kind:", gvk.Kind)

	var specDescriptors, statusDescriptors []descriptor
	for _, t := range pkgTypes {
		// fmt.Printf("\ntype %s\n", t.Name.Name)
		for _, m := range t.Members {
			// fmt.Printf("member %s %s\n", m.Name, m.Type.Name.Name)
			// fmt.Printf("\tcomment lines: %+q\n\n", m.CommentLines)
			specDesc, statusDesc, err := parseCSVGenAnnotations(m, m.CommentLines)
			if err != nil {
				return err
			}
			if specDesc.include {
				setDescriptorDefaultsIfEmpty(&specDesc, m, true)
				specDescriptors = append(specDescriptors, specDesc)
			}
			if statusDesc.include {
				setDescriptorDefaultsIfEmpty(&statusDesc, m, false)
				statusDescriptors = append(statusDescriptors, statusDesc)
			}
		}
	}

	specDescriptors = sortDescriptors(specDescriptors)
	for _, d := range specDescriptors {
		crdDesc.SpecDescriptors = append(crdDesc.SpecDescriptors, olmapiv1alpha1.SpecDescriptor{
			Description:  d.description,
			DisplayName:  d.displayName,
			Path:         d.path,
			XDescriptors: d.xdesc,
		})
	}
	statusDescriptors = sortDescriptors(statusDescriptors)
	for _, d := range statusDescriptors {
		crdDesc.StatusDescriptors = append(crdDesc.StatusDescriptors, olmapiv1alpha1.StatusDescriptor{
			Description:  d.description,
			DisplayName:  d.displayName,
			Path:         d.path,
			XDescriptors: d.xdesc,
		})
	}

	return nil
}

type descriptor struct {
	typ         *types.Type
	include     bool
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

func wrapParseErr(err error) error {
	return errors.Wrap(err, "error parsing csv-gen annotation")
}

// TODO(estroz): apply annotations to all versions or select versions specified by an annotation.
// TODO(estroz): make annotations for all supported customresourcedefinition fields.
func parseCSVGenAnnotations(m types.Member, comments []string) (specDesc, statusDesc descriptor, err error) {
	tags := types.ExtractCommentTags(csvgenPrefix, comments)
	for path, vals := range tags {
		if len(vals) != 1 {
			return specDesc, statusDesc, wrapParseErr(fmt.Errorf("expected one value for %s, got %+q", path, vals))
		}
		val := vals[0]
		// fmt.Printf("path \"%+q\"\n", path)
		pathElems, err := annotations.SplitPath(path)
		if err != nil {
			return specDesc, statusDesc, wrapParseErr(err)
		}
		// fmt.Printf("pathElems \"%+q\"\n", pathElems)
		parentPathElem, childPathElems := pathElems[0], pathElems[1:]
		switch parentPathElem {
		case "customresourcedefinitions":
			switch childPathElems[0] {
			case "specDescriptors":
				err = processDescriptor(&specDesc, childPathElems, val)
				if err != nil {
					return specDesc, statusDesc, wrapParseErr(err)
				}
			case "statusDescriptors":
				err = processDescriptor(&statusDesc, childPathElems, val)
				if err != nil {
					return specDesc, statusDesc, wrapParseErr(err)
				}
			default:
				return specDesc, statusDesc, wrapParseErr(fmt.Errorf(`unsupported %s child path element "%s"`, parentPathElem, childPathElems[0]))
			}
		default:
			return specDesc, statusDesc, wrapParseErr(fmt.Errorf(`unsupported path element "%s"`, parentPathElem))
		}
	}
	return specDesc, statusDesc, nil
}

func processDescriptor(desc *descriptor, pathElems []string, val string) (err error) {
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
		case "path":
			desc.path, err = strconv.Unquote(val)
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

func setDescriptorDefaultsIfEmpty(desc *descriptor, m types.Member, isSpec bool) {
	desc.typ = m.Type
	desc.description = processDescription(m.CommentLines)
	if desc.displayName == "" {
		desc.displayName = getDisplayName(m.Name)
	}
	if desc.path == "" {
		desc.path = getPathFromJSONTags(m.Tags)
	}
	if len(desc.xdesc) == 0 {
		desc.xdesc = getXDescriptorByPath(desc.path, isSpec)
	}
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
