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

package k8sutil

import (
	"reflect"
	"testing"
)

func TestSortVersions(t *testing.T) {
	cases := []struct {
		inputVersions []string
		expected      []string
	}{
		{[]string{""}, []string{""}},
		{[]string{"v1"}, []string{"v1"}},
		{[]string{"v1alpha1"}, []string{"v1alpha1"}},
		{[]string{"v1alpha1", "v1"}, []string{"v1", "v1alpha1"}},
		{
			[]string{"v1alpha1", "v1", "v2", "v2alpha4"},
			[]string{"v2", "v1", "v2alpha4", "v1alpha1"},
		},
		{
			[]string{"v1alpha1", "v1", "v2beta4", "v2alpha4"},
			[]string{"v1", "v2beta4", "v2alpha4", "v1alpha1"},
		},
		{
			[]string{"foo1", "foo10", "foo2", "foo13", "foo52", "foo23", "foo32", "foo33", "foo100"},
			[]string{"foo1", "foo10", "foo100", "foo13", "foo2", "foo23", "foo32", "foo33", "foo52"},
		},
		{
			[]string{"v3beta1", "v12alpha1", "v12alpha2", "v10beta3", "v1", "v11alpha2", "foo1", "v10", "v2", "foo10", "v11beta2"},
			[]string{"v10", "v2", "v1", "v11beta2", "v10beta3", "v3beta1", "v12alpha2", "v12alpha1", "v11alpha2", "foo1", "foo10"},
		},
	}

	for _, c := range cases {
		inputs := make([]string, len(c.inputVersions))
		copy(inputs, c.inputVersions)
		SortVersions(inputs, GetStringSliceName)
		if !reflect.DeepEqual(inputs, c.expected) {
			t.Errorf("not equal:\noutput:   %+q\nexpected: %+q", inputs, c.expected)
		}
	}
}

func TestSortVersionsMajor(t *testing.T) {
	cases := []struct {
		inputVersions []string
		expected      []string
	}{
		{[]string{""}, []string{""}},
		{[]string{"v1"}, []string{"v1"}},
		{[]string{"v1alpha1"}, []string{"v1alpha1"}},
		{[]string{"v1alpha1", "v1"}, []string{"v1", "v1alpha1"}},
		{
			[]string{"v1alpha1", "v1", "v2", "v2alpha4"},
			[]string{"v2", "v2alpha4", "v1", "v1alpha1"},
		},
		{
			[]string{"v1alpha1", "v1", "v2beta4", "v2alpha4", "v3alpha1", "v2"},
			[]string{"v3alpha1", "v2", "v2beta4", "v2alpha4", "v1", "v1alpha1"},
		},
		{
			[]string{"foo1", "foo10", "foo2", "foo13", "foo52", "foo23", "foo32", "foo33", "foo100"},
			[]string{"foo1", "foo10", "foo100", "foo13", "foo2", "foo23", "foo32", "foo33", "foo52"},
		},
		{
			[]string{"v3beta1", "v12alpha1", "v12alpha2", "v10beta3", "v1", "v11alpha2", "foo1", "v10", "v2", "foo10", "v11beta2"},
			[]string{"v12alpha2", "v12alpha1", "v11beta2", "v11alpha2", "v10", "v10beta3", "v3beta1", "v2", "v1", "foo1", "foo10"},
		},
	}

	for _, c := range cases {
		inputs := make([]string, len(c.inputVersions))
		copy(inputs, c.inputVersions)
		SortVersionsMajor(inputs, GetStringSliceName)
		if !reflect.DeepEqual(inputs, c.expected) {
			t.Errorf("not equal:\noutput:   %+q\nexpected: %+q", inputs, c.expected)
		}
	}
}
