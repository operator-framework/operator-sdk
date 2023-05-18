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
	"math/rand"
	"reflect"
	"sort"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"sigs.k8s.io/controller-tools/pkg/crd"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/markers"
	kbutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

var _ = Describe("getTypedDescriptors", func() {
	var (
		markedFields map[crd.TypeIdent][]*fieldInfo
		out          []interface{}
	)

	BeforeEach(func() {
		markedFields = make(map[crd.TypeIdent][]*fieldInfo)
	})

	It("handles an empty set of marked fields", func() {
		out = getTypedDescriptors(markedFields, reflect.TypeOf(v1alpha1.SpecDescriptor{}), spec)
		Expect(out).To(BeEmpty())
	})
	It("returns one spec descriptor for one spec marker on a field", func() {
		markedFields[crd.TypeIdent{}] = []*fieldInfo{
			{
				FieldInfo: markers.FieldInfo{
					Markers: markers.MarkerValues{
						"": []interface{}{
							Descriptor{"spec", "Foo", []string{"urn:alm:descriptor:com.tectonic.ui:text"}, nil},
						},
					},
				},
			},
		}
		out = getTypedDescriptors(markedFields, reflect.TypeOf(v1alpha1.SpecDescriptor{}), spec)
		Expect(out).To(HaveLen(1))
		Expect(out).To(BeEquivalentTo([]interface{}{
			v1alpha1.SpecDescriptor{
				DisplayName:  "Foo",
				XDescriptors: []string{"urn:alm:descriptor:com.tectonic.ui:text"},
			},
		}))
	})
	It("returns no spec descriptors for one status marker on a field", func() {
		markedFields[crd.TypeIdent{}] = []*fieldInfo{
			{
				FieldInfo: markers.FieldInfo{
					Markers: markers.MarkerValues{
						"": []interface{}{
							Descriptor{"status", "Foo", []string{"urn:alm:descriptor:com.tectonic.ui:text"}, nil},
						},
					},
				},
			},
		}
		out = getTypedDescriptors(markedFields, reflect.TypeOf(v1alpha1.SpecDescriptor{}), spec)
		Expect(out).To(BeEmpty())
	})
	It("returns one status descriptor for one status marker on a field", func() {
		markedFields[crd.TypeIdent{}] = []*fieldInfo{
			{
				FieldInfo: markers.FieldInfo{
					Markers: markers.MarkerValues{
						"": []interface{}{
							Descriptor{"status", "Foo", []string{"urn:alm:descriptor:com.tectonic.ui:text"}, nil},
						},
					},
				},
			},
		}
		out = getTypedDescriptors(markedFields, reflect.TypeOf(v1alpha1.StatusDescriptor{}), status)
		Expect(out).To(HaveLen(1))
		Expect(out).To(BeEquivalentTo([]interface{}{
			v1alpha1.StatusDescriptor{
				DisplayName:  "Foo",
				XDescriptors: []string{"urn:alm:descriptor:com.tectonic.ui:text"},
			},
		}))
	})
	It("returns one spec descriptor for three spec markers and one status marker on a field", func() {
		markedFields[crd.TypeIdent{}] = []*fieldInfo{
			{
				FieldInfo: markers.FieldInfo{
					Markers: markers.MarkerValues{
						"": []interface{}{
							Descriptor{"spec", "Foo", nil, nil},
							Descriptor{"spec", "", nil, intPtr(2)},
							Descriptor{"spec", "", []string{"urn:alm:descriptor:com.tectonic.ui:text"}, nil},
							Descriptor{"status", "", []string{"urn:alm:descriptor:com.tectonic.ui:arrayFieldGroup:blah"}, nil},
						},
					},
				},
				pathSegments: []string{"foo", inlinedTag, "bar", "baz"},
			},
		}
		out = getTypedDescriptors(markedFields, reflect.TypeOf(v1alpha1.SpecDescriptor{}), spec)
		Expect(out).To(HaveLen(1))
		Expect(out).To(BeEquivalentTo([]interface{}{
			v1alpha1.SpecDescriptor{
				DisplayName:  "Foo",
				XDescriptors: []string{"urn:alm:descriptor:com.tectonic.ui:text"},
				Path:         "foo.bar.baz",
			},
		}))
	})
	It("returns two spec descriptor for spec markers on two different fields", func() {
		markedFields[crd.TypeIdent{}] = []*fieldInfo{
			{
				FieldInfo: markers.FieldInfo{
					Markers: markers.MarkerValues{
						"": []interface{}{Descriptor{"spec", "Foo", nil, intPtr(1)}},
					},
				},
				pathSegments: []string{"foo"},
			},
			{
				FieldInfo: markers.FieldInfo{
					Markers: markers.MarkerValues{
						"": []interface{}{Descriptor{"spec", "Bar", nil, intPtr(0)}},
					},
				},
				pathSegments: []string{"bar"},
			},
		}
		out = getTypedDescriptors(markedFields, reflect.TypeOf(v1alpha1.SpecDescriptor{}), spec)
		Expect(out).To(HaveLen(2))
		Expect(out).To(BeEquivalentTo([]interface{}{
			v1alpha1.SpecDescriptor{DisplayName: "Bar", Path: "bar"},
			v1alpha1.SpecDescriptor{DisplayName: "Foo", Path: "foo"},
		}))
	})
	It("returns multiple sorted spec descriptors with all orders set", func() {
		markedFields, expected := makeMockMarkedFields()
		out = getTypedDescriptors(markedFields, reflect.TypeOf(v1alpha1.SpecDescriptor{}), spec)
		Expect(out).To(HaveLen(len(expected)))
		Expect(out).To(BeEquivalentTo(expected))
	})
})

func intPtr(i int) *int { return &i }

// makeMockMarkedFields returns a randomly generated mock marked field set,
// and the expected sorted set of descriptors.
func makeMockMarkedFields() (markedFields map[crd.TypeIdent][]*fieldInfo, expected []interface{}) {
	descBuckets := make(map[int][]v1alpha1.SpecDescriptor, 100)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	markedFields = make(map[crd.TypeIdent][]*fieldInfo, 100)
	for i := 0; i < 100; i++ {
		s, err := kbutil.RandomSuffix()
		if err != nil {
			panic(err)
		}
		caser := cases.Title(language.AmericanEnglish)
		name := caser.String(s)
		order := r.Int() % 200 // Very likely to get one conflict.
		ident := crd.TypeIdent{Package: &loader.Package{}, Name: name}
		if _, hasName := markedFields[ident]; hasName {
			continue
		}
		markedFields[ident] = []*fieldInfo{
			{
				FieldInfo: markers.FieldInfo{
					Markers: markers.MarkerValues{
						"": []interface{}{Descriptor{"spec", name, nil, intPtr(order)}},
					},
				},
				pathSegments: []string{s},
			},
		}
		descBuckets[order] = append(descBuckets[order], v1alpha1.SpecDescriptor{DisplayName: name, Path: s})
	}

	orders := make([]int, 0, 100)
	for order := range descBuckets {
		orders = append(orders, order)
	}
	sort.Ints(orders)

	for _, order := range orders {
		bucket := descBuckets[order]
		sort.Slice(bucket, func(i, j int) bool { return bucket[i].DisplayName < bucket[j].DisplayName })
		for _, d := range bucket {
			expected = append(expected, d)
		}
	}

	return markedFields, expected
}
