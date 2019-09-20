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
	"sort"
	"testing"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-sdk/internal/annotations"
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
			"Empty resource string without quotes",
			"", v1alpha1.APIResourceReference{}, true,
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

func cmpDescriptors(a, b descriptor) bool {
	if len(a.xdescs) != len(b.xdescs) {
		return false
	}
	sort.Strings(a.xdescs)
	sort.Strings(b.xdescs)
	for i := range a.xdescs {
		if a.xdescs[i] != b.xdescs[i] {
			return false
		}
	}
	return a.descType == b.descType && a.description == b.description &&
		a.displayName == b.displayName && a.include == b.include &&
		a.path == b.path
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
			descriptor{displayName: "Some display name"},
			false,
		},
		{
			"Descriptor with x-descriptors",
			[]string{"specDescriptors", "x-descriptors"}, `"x:descriptor:ui:hint"`,
			descriptor{xdescs: []string{"x:descriptor:ui:hint"}},
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
			"Descriptor string with uknown path element",
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
			if !cmpDescriptors(c.exp, output) {
				t.Errorf("%s: expected %v, got %v", c.description, c.exp, output)
			}
		}
	}
}

func cmpParsedDescriptions(a, b parsedCRDDescriptions) bool {
	if len(a.descriptors) != len(b.descriptors) {
		return false
	}
	ad, bd := sortDescriptors(a.descriptors), sortDescriptors(b.descriptors)
	for i := range ad {
		if !cmpDescriptors(ad[i], bd[i]) {
			return false
		}
	}
	if len(a.resources) != len(b.resources) {
		return false
	}
	ar, br := sortResources(a.resources), sortResources(b.resources)
	for i := range ar {
		if ar[i] != br[i] {
			return false
		}
	}
	return a.displayName == b.displayName
}

func TestParseCSVGenAnnotations(t *testing.T) {
	crdPath := annotations.JoinPrefix(csvgenPrefix, "customresourcedefinitions")
	specDescPath := annotations.JoinPath(crdPath, "specDescriptors")
	statusDescPath := annotations.JoinPath(crdPath, "statusDescriptors")
	displayNamePath := annotations.JoinPath(crdPath, "displayName")
	resourcesPath := annotations.JoinPath(crdPath, "resources")
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
			parsedCRDDescriptions{displayName: "foo"},
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
			},
			false,
		},
		{
			"Comment on type member with spec inclusion annotation",
			[]string{
				annotations.JoinAnnotation(specDescPath, "true"),
			},
			parsedCRDDescriptions{
				descriptors: []descriptor{{include: true, descType: typeSpec}},
			},
			false,
		},
		{
			"Comment on type member with one spec annotation and no spec inclusion annotation",
			[]string{
				annotations.JoinAnnotation(annotations.JoinPath(specDescPath, "displayName"), `"foo"`),
			},
			parsedCRDDescriptions{},
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
				descriptors: []descriptor{{include: true, descType: typeSpec, displayName: "foo"}},
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
					{include: true, descType: typeSpec, displayName: "foo", xdescs: []string{"some:ui:hint"}},
					{include: true, descType: typeStatus, displayName: "foo", xdescs: []string{"some:ui:hint"}},
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
			if !cmpParsedDescriptions(c.exp, output) {
				t.Errorf("%s: expected %v, got %v", c.description, c.exp, output)
			}
		}
	}
}
