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
	"math"
	"reflect"
	"sort"
	"strings"

	"github.com/fatih/structtag"
	"github.com/markbates/inflect"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"sigs.k8s.io/controller-tools/pkg/crd"
	crdmarkers "sigs.k8s.io/controller-tools/pkg/crd/markers"
	"sigs.k8s.io/controller-tools/pkg/loader"
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
func (g generator) buildCRDDescriptionFromType(gvk schema.GroupVersionKind, typeIdent crd.TypeIdent, kindType *markers.TypeInfo) (v1alpha1.CRDDescription, int, error) {

	// Initialize the description.
	description := v1alpha1.CRDDescription{
		Description: kindType.Doc,
		DisplayName: k8sutil.GetDisplayName(gvk.Kind),
		Version:     gvk.Version,
		Kind:        gvk.Kind,
	}

	// Parse resources and displayName from the kind type's markers.
	descriptionOrder := math.MaxInt32
	for _, markers := range kindType.Markers {
		for _, marker := range markers {
			switch d := marker.(type) {
			case Description:
				if d.Order != nil {
					descriptionOrder = *d.Order
				}
				if d.DisplayName != "" {
					description.DisplayName = d.DisplayName
				}
				if len(d.Resources) != 0 {
					refs, err := d.Resources.toResourceReferences()
					if err != nil {
						return v1alpha1.CRDDescription{}, 0, err
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
	sortResources(description.Resources)

	specDescriptors, err := g.getTypedDescriptors(typeIdent.Package, kindType, reflect.TypeOf(v1alpha1.SpecDescriptor{}), spec)
	if err != nil {
		return v1alpha1.CRDDescription{}, 0, err
	}
	for _, d := range specDescriptors {
		description.SpecDescriptors = append(description.SpecDescriptors, d.(v1alpha1.SpecDescriptor))
	}

	statusDescriptors, err := g.getTypedDescriptors(typeIdent.Package, kindType, reflect.TypeOf(v1alpha1.StatusDescriptor{}), status)
	if err != nil {
		return v1alpha1.CRDDescription{}, 0, err
	}
	for _, d := range statusDescriptors {
		description.StatusDescriptors = append(description.StatusDescriptors, d.(v1alpha1.StatusDescriptor))
	}

	return description, descriptionOrder, nil
}

func (g generator) getTypedDescriptors(pkg *loader.Package, kindType *markers.TypeInfo, t reflect.Type, descType string) ([]interface{}, error) {
	// Find child in the kind type.
	child, err := findChildForDescType(kindType, descType)
	if err != nil {
		return nil, err
	}

	// Find annotated fields of child and parse them into descriptors.
	markedFields, err := g.getMarkedChildrenOfField(pkg, child)
	if err != nil {
		return nil, err
	}

	return getTypedDescriptors(markedFields, t, descType), nil
}

func getTypedDescriptors(markedFields map[crd.TypeIdent][]*fieldInfo, t reflect.Type, descType string) (descriptors []interface{}) {
	descriptorBuckets := make(map[int][]reflect.Value)
	orders := make([]int, 0)
	for _, fields := range markedFields {
		for _, field := range fields {
			v := reflect.New(t)
			if order, include := field.setDescriptorFields(v, descType); include {
				descriptorBuckets[order] = append(descriptorBuckets[order], reflect.Indirect(v))
				orders = append(orders, order)
			}
		}
	}
	sort.Ints(orders)

	descriptorVals := make([]reflect.Value, 0)
	for _, order := range orders {
		if bucket, hasOrder := descriptorBuckets[order]; hasOrder {
			sortDescriptors(bucket)
			descriptorVals = append(descriptorVals, bucket...)
			delete(descriptorBuckets, order)
		}
	}

	for _, v := range descriptorVals {
		descriptors = append(descriptors, v.Interface())
	}

	return descriptors
}

// findChildForDescType returns a field with a tag matching string(typ) by searching all top-level fields in info.
// If no field is found, an error is returned.
func findChildForDescType(info *markers.TypeInfo, descType string) (markers.FieldInfo, error) {
	for _, field := range info.Fields {
		tags, err := structtag.Parse(string(field.Tag))
		if err != nil {
			return markers.FieldInfo{}, err
		}
		jsonTag, err := tags.Get("json")
		if err == nil && jsonTag.Name == descType {
			return field, nil
		}
	}
	return markers.FieldInfo{}, fmt.Errorf("no %s found for type %s", descType, info.Name)
}

// sortDescriptors sorts a slice of structs with a Path field by comparing Path strings naturally.
func sortDescriptors(values []reflect.Value) {
	sort.Slice(values, func(i, j int) bool {
		return values[i].FieldByName("Path").String() < values[j].FieldByName("Path").String()
	})
}

// sortResources sorts a slice of structs with Name, Kind, and Version fields
// by comparing those field's strings in natural order.
func sortResources(rs []v1alpha1.APIResourceReference) {
	sort.Slice(rs, func(i, j int) bool {
		if rs[i].Name == rs[j].Name {
			if rs[i].Kind == rs[j].Kind {
				return version.CompareKubeAwareVersionStrings(rs[i].Version, rs[j].Version) > 0
			}
			return rs[i].Kind < rs[j].Kind
		}
		return rs[i].Name < rs[j].Name
	})
}
