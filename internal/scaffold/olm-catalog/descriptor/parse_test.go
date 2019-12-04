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
	"reflect"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/annotations"
	"k8s.io/gengo/types"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
)

func TestParseResource(t *testing.T) {
	cases := []struct {
		description string
		input       string
		exp         v1alpha1.APIResourceReference
		wantErr     bool
	}{
		{
			"Resource string no name",
			`"Memcached,v1"`,
			v1alpha1.APIResourceReference{Kind: "Memcached", Version: "v1"},
			false,
		},
		{
			"Resource string with name",
			`"Memcached,v1,\"memcached.example.com\""`,
			v1alpha1.APIResourceReference{Kind: "Memcached", Version: "v1", Name: "memcached.example.com"},
			false,
		},
		{
			"Resource string literal with name",
			"`Memcached,v1,\"memcached.example.com\"`",
			v1alpha1.APIResourceReference{Kind: "Memcached", Version: "v1", Name: "memcached.example.com"},
			false,
		},
		{
			"Empty resource string without quotes",
			``, v1alpha1.APIResourceReference{}, true,
		},
		{
			"Empty resource string with quotes",
			`""`, v1alpha1.APIResourceReference{}, true,
		},
		{
			"Resource string with no version",
			`"Memcached"`, v1alpha1.APIResourceReference{}, true,
		},
		{
			"Resource string with unquoted name",
			`"Memcached,v1,memcached.example.com"`, v1alpha1.APIResourceReference{}, true,
		},
		{
			"Resource string literal with unquoted name",
			"`Memcached,v1,memcached.example.com`", v1alpha1.APIResourceReference{}, true,
		},
	}

	for _, c := range cases {
		output, err := parseResource(c.input)
		if err != nil {
			if !c.wantErr {
				t.Errorf("%s: expected nil error, got %q", c.description, err)
			}
			continue
		} else if c.wantErr {
			t.Errorf("%s: expected non-nil error, got nil error", c.description)
			continue
		}

		if !c.wantErr {
			if c.exp != output {
				t.Errorf("%s: expected %s, got %s", c.description, c.exp, output)
			}
		}
	}
}

func TestParseDescriptor(t *testing.T) {
	cases := []struct {
		description string
		pathElems   []string
		val         string
		exp         descriptor
		wantErr     bool
	}{
		{
			"Descriptor with true",
			[]string{"specDescriptors"}, "true",
			descriptor{include: true},
			false,
		},
		{
			"Descriptor with false",
			[]string{"specDescriptors"}, "false",
			descriptor{include: false},
			false,
		},
		{
			"Descriptor with displayName",
			[]string{"specDescriptors", "displayName"}, `"Some display name"`,
			descriptor{SpecDescriptor: v1alpha1.SpecDescriptor{DisplayName: "Some display name"}},
			false,
		},
		{
			"Descriptor with x-descriptors",
			[]string{"specDescriptors", "x-descriptors"}, `"x:descriptor:ui:hint"`,
			descriptor{SpecDescriptor: v1alpha1.SpecDescriptor{XDescriptors: []string{"x:descriptor:ui:hint"}}},
			false,
		},
		{
			"Descriptor with non-boolean",
			[]string{"specDescriptors"}, "foo", descriptor{},
			true,
		},
		{
			"Empty descriptor path elements",
			[]string{}, "", descriptor{},
			true,
		},
		{
			"Descriptor with too many path elements",
			[]string{"a", "b", "c"}, "", descriptor{},
			true,
		},
		{
			"Descriptor string with unknown path element",
			[]string{"specDescriptors", "bar"}, "", descriptor{},
			true,
		},
	}

	for _, c := range cases {
		output := descriptor{}
		err := parseMemberAnnotation(&output, c.pathElems, c.val)
		if err != nil {
			if !c.wantErr {
				t.Errorf("%s: expected nil error, got %q", c.description, err)
			}
			continue
		} else if c.wantErr {
			t.Errorf("%s: expected non-nil error, got nil error", c.description)
			continue
		}

		if !c.wantErr {
			if !reflect.DeepEqual(c.exp, output) {
				t.Errorf("%s: expected %v, got %v", c.description, c.exp, output)
			}
		}
	}
}

func TestParseCSVGenAnnotations(t *testing.T) {
	crdPath := annotations.JoinPrefix(csvgenPrefix, "customresourcedefinitions")
	specDescPath := annotations.JoinPath(crdPath, "specDescriptors")
	statusDescPath := annotations.JoinPath(crdPath, "statusDescriptors")
	displayNamePath := annotations.JoinPath(crdPath, "displayName")
	resourcesPath := annotations.JoinPath(crdPath, "resources")
	emptyDescriptors := []descriptor{{descType: typeSpec}, {descType: typeStatus}}

	cases := []struct {
		description string
		comments    []string
		exp         parsedCRDDescriptions
		wantErr     bool
	}{
		{
			"Comment on type with one annotation",
			[]string{
				annotations.JoinAnnotation(displayNamePath, `"foo"`),
			},
			parsedCRDDescriptions{displayName: "foo", descriptors: emptyDescriptors},
			false,
		},
		{
			"Comment on type with description and all annotations",
			[]string{
				annotations.JoinAnnotation(displayNamePath, `"foo"`),
				annotations.JoinAnnotation(resourcesPath, `"Pod,v1,\"pod\""`),
				annotations.JoinAnnotation(resourcesPath, `"Deployment,v1"`),
				annotations.JoinAnnotation(resourcesPath, `"Service,v1,\"some.example.service.com\""`),
			},
			parsedCRDDescriptions{
				displayName: "foo",
				resources: []v1alpha1.APIResourceReference{
					{Kind: "Pod", Version: "v1", Name: "pod"},
					{Kind: "Deployment", Version: "v1"},
					{Kind: "Service", Version: "v1", Name: "some.example.service.com"},
				},
				descriptors: emptyDescriptors,
			},
			false,
		},
		{
			"Comment on type member with spec inclusion annotation",
			[]string{
				annotations.JoinAnnotation(specDescPath, "true"),
			},
			parsedCRDDescriptions{
				descriptors: []descriptor{{include: true, descType: typeSpec}, {descType: typeStatus}},
			},
			false,
		},
		{
			"Comment on type member with one spec annotation and no spec inclusion annotation",
			[]string{
				annotations.JoinAnnotation(annotations.JoinPath(specDescPath, "displayName"), `"foo"`),
			},
			parsedCRDDescriptions{
				descriptors: []descriptor{
					{SpecDescriptor: v1alpha1.SpecDescriptor{DisplayName: "foo"}, descType: typeSpec},
					{include: false, descType: typeStatus},
				},
			},
			false,
		},
		{
			"Comment on type member with one spec and status annotation and no status inclusion annotation",
			[]string{
				annotations.JoinAnnotation(specDescPath, "true"),
				annotations.JoinAnnotation(annotations.JoinPath(specDescPath, "displayName"), `"foo"`),
				annotations.JoinAnnotation(annotations.JoinPath(statusDescPath, "displayName"), `"foo"`),
			},
			parsedCRDDescriptions{
				descriptors: []descriptor{
					{include: true, descType: typeSpec, SpecDescriptor: v1alpha1.SpecDescriptor{DisplayName: "foo"}},
					{include: false, descType: typeStatus, SpecDescriptor: v1alpha1.SpecDescriptor{DisplayName: "foo"}},
				},
			},
			false,
		},
		{
			"Comment on type member with spec and status annotations and both inclusion annotations",
			[]string{
				annotations.JoinAnnotation(specDescPath, "true"),
				annotations.JoinAnnotation(statusDescPath, "true"),
				annotations.JoinAnnotation(annotations.JoinPath(specDescPath, "displayName"), `"foo"`),
				annotations.JoinAnnotation(annotations.JoinPath(specDescPath, "x-descriptors"), `"some:ui:hint"`),
				annotations.JoinAnnotation(annotations.JoinPath(statusDescPath, "displayName"), `"foo"`),
				annotations.JoinAnnotation(annotations.JoinPath(statusDescPath, "x-descriptors"), `"some:ui:hint"`),
			},
			parsedCRDDescriptions{
				descriptors: []descriptor{
					{include: true, descType: typeSpec, SpecDescriptor: v1alpha1.SpecDescriptor{DisplayName: "foo", XDescriptors: []string{"some:ui:hint"}}},
					{include: true, descType: typeStatus, SpecDescriptor: v1alpha1.SpecDescriptor{DisplayName: "foo", XDescriptors: []string{"some:ui:hint"}}},
				},
			},
			false,
		},
		{
			"Comment on type with unknown path element",
			[]string{
				annotations.JoinAnnotation(annotations.JoinPath(annotations.JoinPrefix(csvgenPrefix, "unknown"), "resources"), `"Deployment,v1"`),
			},
			parsedCRDDescriptions{},
			true,
		},
		{
			"Comment on type with unknown child path element",
			[]string{
				annotations.JoinAnnotation(annotations.JoinPath(crdPath, "unknown"), `"Deployment,v1"`),
			},
			parsedCRDDescriptions{},
			true,
		},
		{
			"Comment on type with a bad displayName annotation",
			[]string{
				annotations.JoinAnnotation(displayNamePath, `foo`),
			},
			parsedCRDDescriptions{},
			true,
		},
	}

	for _, c := range cases {
		output, err := parseCSVGenAnnotations(c.comments)
		if !c.wantErr && err != nil {
			t.Errorf("%s: expected nil error, got %q", c.description, err)
		} else if c.wantErr && err == nil {
			t.Errorf("%s: expected non-nil error, got nil error", c.description)
		} else if !c.wantErr && err == nil {
			if !reflect.DeepEqual(c.exp, output) {
				t.Errorf("%s:\nexpected\n\t%v\ngot\n\t%v", c.description, c.exp, output)
			}
		}
	}
}

func TestParsePathFromMember(t *testing.T) {
	cases := []struct {
		description string
		member      types.Member
		exp         string
		wantErr     bool
	}{
		{"empty tag", types.Member{}, "Foo", false},
		{"valid single tag", types.Member{Tags: `json:"foo"`}, "foo", false},
		{"valid single with omitempty tag", types.Member{Tags: `json:"foo,omitempty"`}, "foo", false},
		{"valid empty with omitempty tag", types.Member{Tags: `json:",omitempty"`}, "Foo", false},
		{"valid single with inline tag", types.Member{Tags: `json:"foo,inline"`}, inlinedTag, false},
		{"valid empty with inline tag", types.Member{Tags: `json:",inline"`}, inlinedTag, false},
		{"valid ignore tag", types.Member{Tags: `json:"-"`}, ignoredTag, false},
		{"valid hyphen as name", types.Member{Tags: `json:"-,"`}, "-", false},
		{"JSON tag in multiple tags", types.Member{Tags: `json:"foo" protobuf:"bar"`}, "foo", false},
		{"no JSON tag in tags", types.Member{Tags: `protobuf:"foo"`}, "Foo", false},
		{"invalid tags", types.Member{Tags: `blahblah`}, "", true},
	}

	for _, c := range cases {
		c.member.Name = "Foo"
		output, err := getPathFromMember(c.member)
		if !c.wantErr && err != nil {
			t.Errorf("%s: expected nil error, got %q", c.description, err)
		} else if c.wantErr && err == nil {
			t.Errorf("%s: expected non-nil error, got nil error", c.description)
		} else if !c.wantErr && err == nil {
			if c.exp != output {
				t.Errorf("%s:\nexpected\n\t%v\ngot\n\t%v", c.description, c.exp, output)
			}
		}
	}
}
