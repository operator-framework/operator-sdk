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
	"testing"
)

func TestLessVersions(t *testing.T) {
	cases := []struct {
		inputTuple   [2]string
		expectedBool bool
	}{
		{[2]string{"", ""}, false},
		{[2]string{"v1", "v1"}, false},
		{[2]string{"v1beta1", "v1beta1"}, false},
		{[2]string{"v1alpha1", "v1alpha1"}, false},
		{[2]string{"v1", ""}, true},
		{[2]string{"", "v1"}, false},
		{[2]string{"v1", "v1beta1"}, true},
		{[2]string{"v1", "v1alpha1"}, true},
		{[2]string{"v1beta1", "v1"}, false},
		{[2]string{"v1alpha1", "v1"}, false},
		{[2]string{"v1beta1", "v1alpha1"}, true},
		{[2]string{"v1alpha1", "v1beta1"}, false},
		{[2]string{"v2alpha1", "v1"}, false},
		{[2]string{"v1", "v2alpha1"}, true},
		{[2]string{"v1", "v1alpha2"}, true},
		{[2]string{"v1", "v10alpha10"}, true},
		{[2]string{"v11", "v10alpha10"}, true},
		{[2]string{"v1", "v2beta1"}, true},
		{[2]string{"v1", "v1beta2"}, true},
		{[2]string{"v1", "v10beta10"}, true},
		{[2]string{"v11", "v10beta10"}, true},
		{[2]string{"v1alpha2000", "v1alpha300"}, true},
		{[2]string{"v2alpha2000", "v1alpha300"}, true},
		{[2]string{"v1alpha2000", "v2alpha300"}, false},
		{[2]string{"foo1", "foo10"}, true},
		{[2]string{"foo1", "foo2"}, true},
		{[2]string{"foo2", "foo10"}, false},
	}

	for _, c := range cases {
		verA, verB := c.inputTuple[0], c.inputTuple[1]
		if v := LessVersions(verA, verB); v != c.expectedBool {
			t.Errorf("Input not ordered as expected for %q vs %q: got %v wanted %v", verA, verB, v, c.expectedBool)
		}
	}
}
