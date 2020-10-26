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
	"sort"
	"strings"

	"github.com/fatih/structtag"
	"github.com/markbates/inflect"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	crdmarkers "sigs.k8s.io/controller-tools/pkg/crd/markers"
	"sigs.k8s.io/controller-tools/pkg/markers"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

// MakeFullGroupFromName returns everything but the first element of a CRD name,
// which by definition is <resource>.<full group>.
func MakeFullGroupFromName(name string) string {
	return getHalfBySep(name, ".", 1)
}

// MakeGroupFromFullGroup returns the first element of an API group, ex. "foo" of "foo.example.com".
func MakeGroupFromFullGroup(group string) string {
	return getHalfBySep(group, ".", 0)
}

// getHalfBySep splits s by the first sep encountered and returns the first
// (half = 0) or second (half = 1) element of the result.
func getHalfBySep(s, sep string, half uint) string {
	if split := strings.SplitN(s, sep, 2); len(split) == 2 && half < 2 {
		return split[half]
	}
	return s
}

// buildCRDDescriptionFromType builds a crdDescription for the Go API defined
// by key from markers and type information in g.types.
func (g generator) buildCRDDescriptionFromType(gvk schema.GroupVersionKind, kindType *markers.TypeInfo) (v1alpha1.CRDDescription, error) {

	// Initialize the description.
	description := v1alpha1.CRDDescription{
		Description: kindType.Doc,
		DisplayName: k8sutil.GetDisplayName(gvk.Kind),
		Version:     gvk.Version,
		Kind:        gvk.Kind,
	}

	// Parse resources and displayName from the kind type's markers.
	for _, markers := range kindType.Markers {
		for _, marker := range markers {
			switch d := marker.(type) {
			case Description:
				if d.DisplayName != "" {
					description.DisplayName = d.DisplayName
				}
				if len(d.Resources) != 0 {
					refs, err := d.Resources.toResourceReferences()
					if err != nil {
						return v1alpha1.CRDDescription{}, err
					}
					description.Resources = append(description.Resources, refs...)
				}
			case crdmarkers.Resource:
				if d.Path != "" {
					description.Name = fmt.Sprintf("%s.%s", d.Path, gvk.Group)
				}
			}
		}
	}
	// The default, if the resource marker's path value is not set, is to use a pluralized form of lowercase kind.
	if description.Name == "" {
		description.Name = fmt.Sprintf("%s.%s", inflect.Pluralize(strings.ToLower(gvk.Kind)), gvk.Group)
	}
	sortDescription(description.Resources)

	// Find spec and status in the kind type.
	spec, err := findChildForDescType(kindType, specDescType)
	if err != nil {
		return v1alpha1.CRDDescription{}, err
	}
	status, err := findChildForDescType(kindType, statusDescType)
	if err != nil {
		return v1alpha1.CRDDescription{}, err
	}

	// Find annotated fields of spec and parse them into specDescriptors.
	markedFields, err := g.getMarkedChildrenOfField(spec)
	if err != nil {
		return v1alpha1.CRDDescription{}, err
	}
	specDescriptors := []v1alpha1.SpecDescriptor{}
	for _, fields := range markedFields {
		for _, field := range fields {
			if descriptor, include := field.toSpecDescriptor(); include {
				specDescriptors = append(specDescriptors, descriptor)
			}
		}
	}
	sortDescriptors(specDescriptors)
	description.SpecDescriptors = specDescriptors

	// Find annotated fields of status and parse them into statusDescriptors.
	markedFields, err = g.getMarkedChildrenOfField(status)
	if err != nil {
		return v1alpha1.CRDDescription{}, err
	}
	statusDescriptors := []v1alpha1.StatusDescriptor{}
	for _, fields := range markedFields {
		for _, field := range fields {
			if descriptor, include := field.toStatusDescriptor(); include {
				statusDescriptors = append(statusDescriptors, descriptor)
			}
		}
	}
	sortDescriptors(statusDescriptors)
	description.StatusDescriptors = statusDescriptors

	return description, nil
}

// findChildForDescType returns a field with a tag matching string(typ) by searching all top-level fields in info.
// If no field is found, an error is returned.
func findChildForDescType(info *markers.TypeInfo, typ descType) (markers.FieldInfo, error) {
	for _, field := range info.Fields {
		tags, err := structtag.Parse(string(field.Tag))
		if err != nil {
			return markers.FieldInfo{}, err
		}
		jsonTag, err := tags.Get("json")
		if err == nil && jsonTag.Name == string(typ) {
			return field, nil
		}
	}
	return markers.FieldInfo{}, fmt.Errorf("no %s found for type %s", typ, info.Name)
}

// sortDescriptors sorts a slice of structs with a Path field by comparing Path strings naturally.
func sortDescriptors(v interface{}) {
	slice := reflect.ValueOf(v)
	values := toValueSlice(slice)
	sort.Slice(values, func(i, j int) bool {
		return values[i].FieldByName("Path").String() < values[j].FieldByName("Path").String()
	})
	for i := 0; i < slice.Len(); i++ {
		slice.Index(i).Set(values[i])
	}
}

// sortDescription sorts a slice of structs with Name, Kind, and Version fields
// by comparing those field's strings in natural order.
func sortDescription(v interface{}) {
	slice := reflect.ValueOf(v)
	values := toValueSlice(slice)
	sort.Slice(values, func(i, j int) bool {
		nameI := values[i].FieldByName("Name").String()
		nameJ := values[j].FieldByName("Name").String()
		if nameI == nameJ {
			kindI := values[i].FieldByName("Kind").String()
			kindJ := values[j].FieldByName("Kind").String()
			if kindI == kindJ {
				versionI := values[i].FieldByName("Version").String()
				versionJ := values[j].FieldByName("Version").String()
				return version.CompareKubeAwareVersionStrings(versionI, versionJ) > 0
			}
			return kindI < kindJ
		}
		return nameI < nameJ
	})
	for i := 0; i < slice.Len(); i++ {
		slice.Index(i).Set(values[i])
	}
}

// toValueSlice creates a slice of values that can be sorted by arbitrary fields.
func toValueSlice(slice reflect.Value) []reflect.Value {
	sliceCopy := reflect.MakeSlice(slice.Type(), slice.Len(), slice.Len())
	reflect.Copy(sliceCopy, slice)
	values := make([]reflect.Value, sliceCopy.Len())
	for i := 0; i < sliceCopy.Len(); i++ {
		values[i] = sliceCopy.Index(i)
	}
	return values
}
