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
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/fatih/structtag"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	crdmarkers "sigs.k8s.io/controller-tools/pkg/crd/markers"
	"sigs.k8s.io/controller-tools/pkg/markers"

	sdkmarkers "github.com/operator-framework/operator-sdk/internal/markers"
	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

const (
	// csvPrefix is the prefix for all csv marker names.
	csvPrefix = sdkmarkers.Prefix + ":csv"
	// crdMarkerName is the base marker name for all customresourcedefinitions markers.
	crdMarkerName = csvPrefix + ":customresourcedefinitions"
)

// +operator-sdk:csv:customresourcedefinitions:displayName="string",resources={ {kind,version,name} , ... }
var typeDefinition = markers.Must(markers.MakeDefinition(crdMarkerName, markers.DescribesType, Description{}))

// +operator-sdk:csv:customresourcedefinitions:type=<spec|status>,displayName="name",xDescriptors="ui:elements:foo:bar"
var fieldDefinition = markers.Must(markers.MakeDefinition(crdMarkerName, markers.DescribesField, Descriptor{}))

// See https://github.com/kubernetes-sigs/controller-tools/blob/92e95c1/pkg/crd/markers/crd.go#L40
var crdResourceDefinition = markers.Must(markers.MakeDefinition("kubebuilder:resource", markers.DescribesType, crdmarkers.Resource{}))

// registerMarkers adds type and field marker definitions to a registry.
func registerMarkers(into *markers.Registry) error {
	if err := into.Register(typeDefinition); err != nil {
		return fmt.Errorf("error registering type definition: %v", err)
	}
	into.AddHelp(typeDefinition, Description{}.Help())
	if err := into.Register(fieldDefinition); err != nil {
		return fmt.Errorf("error registering field definition: %v", err)
	}
	into.AddHelp(fieldDefinition, Descriptor{}.Help())

	// External definitions.
	if err := into.Register(crdResourceDefinition); err != nil {
		return fmt.Errorf("error registering CRD resource definition: %v", err)
	}
	into.AddHelp(crdResourceDefinition, crdmarkers.Resource{}.Help())
	return nil
}

// Resource is a list of strings defining a CRD description resource.
type Resource []string

// Resources is a list of resource definitions.
type Resources []Resource

// +controllertools:marker:generateHelp:category=Description

// Description is a type used to receive type-level CRD description markers.
type Description struct {
	// Resources is a list of string lists, each of which defines a CRD description resource. The marker format is:
	// { { "kind" , "version" ( , "name")? } , ... }
	Resources Resources `marker:",optional"`
	// DisplayName is the displayName of a CRD description.
	DisplayName string `marker:",optional"`
}

// +controllertools:marker:generateHelp:category=Descriptor

// Descriptor is a type used to receive field-level spec and status descriptor markers. Format of marker:
type Descriptor struct {
	// Type is one of: "spec", "status".
	Type string `marker:",optional"`
	// DisplayName is the displayName of a spec or status description.
	DisplayName string `marker:",optional"`
	// XDescriptors is a list of UI path strings. The marker format is:
	// "ui:element:foo,ui:element:bar"
	XDescriptors []string `marker:",optional"`
}

// toResourceReferences transforms Resources into a apiResourceReference slice.
func (resources Resources) toResourceReferences() (rs []v1alpha1.APIResourceReference, err error) {
	for _, resource := range resources {
		if l := len(resource); l < 2 {
			return nil, fmt.Errorf("resource %+q did not have at least a kind and a version", resource)
		}
		r := v1alpha1.APIResourceReference{
			Kind:    strings.TrimSpace(resource[0]),
			Version: strings.TrimSpace(resource[1]),
		}
		if len(resource) == 3 {
			r.Name = strings.TrimSpace(resource[2])
		}
		rs = append(rs, r)
	}
	return rs, nil
}

// fieldInfo is a markers.FieldInfo wrapper that also holds path segments.
type fieldInfo struct {
	markers.FieldInfo
	pathSegments []string
}

// descType is a string identifying a descriptor type.
type descType string

const (
	specDescType   descType = "spec"
	statusDescType descType = "status"
)

// toStatusDescriptor transforms a fieldInfo into a specDescriptor.
func (fi fieldInfo) toSpecDescriptor() (descriptor v1alpha1.SpecDescriptor, include bool) {
	include = fi.setDescriptorFields(&descriptor, specDescType)
	return
}

// toStatusDescriptor transforms a fieldInfo into a statusDescriptor.
func (fi fieldInfo) toStatusDescriptor() (descriptor v1alpha1.StatusDescriptor, include bool) {
	include = fi.setDescriptorFields(&descriptor, statusDescType)
	return
}

// setDescriptorFields sets a struct with Description, Path, DisplayName, and XDescriptors fields by reflection.
func (fi fieldInfo) setDescriptorFields(d interface{}, typ descType) bool {
	path, include := makePath(fi.pathSegments)
	if !include {
		return false
	}

	seenDescType := false
	displayName, xDescriptors := "", []string{}
	for _, markers := range fi.Markers {
		for _, marker := range markers {
			d, isDescriptor := marker.(Descriptor)
			if isDescriptor && d.Type == string(typ) {
				if d.DisplayName != "" && displayName == "" {
					displayName = d.DisplayName
				}
				xDescriptors = append(xDescriptors, d.XDescriptors...)
				seenDescType = true
			}
		}
	}
	if displayName == "" {
		displayName = k8sutil.GetDisplayName(fi.Name)
	}

	v := reflect.ValueOf(d)
	v.Elem().FieldByName("Description").SetString(fi.Doc)
	v.Elem().FieldByName("Path").SetString(path)
	v.Elem().FieldByName("DisplayName").SetString(displayName)
	v.Elem().FieldByName("XDescriptors").Set(reflect.ValueOf(xDescriptors))

	return seenDescType
}

// makePath creates a path string from raw path segments. These segments can encode extra information
// about what field it came from. If a path should be ignored by the caller, it returns false.
func makePath(rawSegments []string) (string, bool) {
	pathSegments := []string{}
	for i, segment := range rawSegments {
		switch {
		case segment == ignoredTag:
			// Ignored fields are not serialized and therefore its own path segment
			// and those of its children should not be included in the path.
			return "", false
		case segment == inlinedTag:
			// Inlined struct types move their fields into their parents, so the path segment
			// of such a field should not be in the path if it is last in the path.
			if i == len(rawSegments)-1 {
				return "", false
			}
			continue
		case strings.HasSuffix(segment, "[0]") && i == len(rawSegments)-1:
			// Only include an arrayFieldGroup suffix if there is a child path segment.
			segment = strings.TrimSuffix(segment, "[0]")
		}
		pathSegments = append(pathSegments, segment)
	}
	return strings.Join(pathSegments, "."), true
}

const (
	inlinedTag = "##inline##"
	ignoredTag = "##ignore##"
)

// getPathSegmentForField parses a path segment from a field's tag.
func getPathSegmentForField(finfo markers.FieldInfo) (string, error) {
	// Embedded fields are inlined and children may be included.
	if len(finfo.RawField.Names) == 0 {
		return inlinedTag, nil
	}
	// Unexported fields should be ignored in downstream processing.
	if !isExported(finfo.Name) {
		return ignoredTag, nil
	}
	tags, err := structtag.Parse(string(finfo.Tag))
	if err != nil {
		return "", err
	}
	jsonTag, err := tags.Get("json")
	if err == nil {
		// Parse returns an error if no JSON tag is in tags, at which point we'll use another method to get path.
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
	// There is no JSON tag in tags or tag name is empty. Use info name as path as json.Marshal does.
	return finfo.Name, nil
}

// isExported returns true if name is an exported struct field name.
func isExported(name string) bool {
	return len(name) != 0 && !unicode.IsLower(rune(name[0]))
}

func contains(options []string, key string) bool {
	for _, opt := range options {
		if opt == key {
			return true
		}
	}
	return false
}
