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
	"sort"
	"testing"

	apiextv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func stringsToCRDVersions(vs []string) (cvs CRDVersions) {
	for _, v := range vs {
		cvs = append(cvs, apiextv1beta1.CustomResourceDefinitionVersion{
			Name: v,
		})
	}
	return cvs
}

func TestSortVersions(t *testing.T) {
	cases := []struct {
		inputVersions CRDVersions
		expected      CRDVersions
	}{
		{stringsToCRDVersions([]string{""}), stringsToCRDVersions([]string{""})},
		{stringsToCRDVersions([]string{"v1"}), stringsToCRDVersions([]string{"v1"})},
		{stringsToCRDVersions([]string{"v1alpha1"}), stringsToCRDVersions([]string{"v1alpha1"})},
		{stringsToCRDVersions([]string{"v1alpha1", "v1"}), stringsToCRDVersions([]string{"v1", "v1alpha1"})},
		{
			stringsToCRDVersions([]string{"foo1", "foo10", "foo2", "foo13", "foo52", "foo23", "foo32", "foo33", "foo100"}),
			stringsToCRDVersions([]string{"foo1", "foo10", "foo100", "foo13", "foo2", "foo23", "foo32", "foo33", "foo52"}),
		},
		{
			stringsToCRDVersions([]string{"v1alpha10", "v1alpha1", "v1alpha2000", "v1alpha3", "v1alpha2", "v1alpha300"}),
			stringsToCRDVersions([]string{"v1alpha2000", "v1alpha300", "v1alpha10", "v1alpha3", "v1alpha2", "v1alpha1"}),
		},
		{
			stringsToCRDVersions([]string{"v3beta1", "v12alpha1", "v12alpha2", "v10beta3", "v1", "v11alpha2", "foo1", "v10", "v2", "foo10", "v11beta2"}),
			stringsToCRDVersions([]string{"v10", "v2", "v1", "v11beta2", "v10beta3", "v3beta1", "v12alpha2", "v12alpha1", "v11alpha2", "foo1", "foo10"}),
		},
	}

	for _, c := range cases {
		sort.Sort(c.inputVersions)
		if !reflect.DeepEqual(c.inputVersions, c.expected) {
			t.Errorf("Output not sorted as expected:\noutput:   %+v\nexpected: %+v", c.inputVersions, c.expected)
		}
	}
}
