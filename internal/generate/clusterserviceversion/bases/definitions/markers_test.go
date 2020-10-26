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
	"reflect"
	"testing"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
)

// TODO(estroz): migrate to ginkgo/gomega

func TestParseResource(t *testing.T) {
	cases := []struct {
		description string
		input       Resources
		exp         []v1alpha1.APIResourceReference
		wantErr     bool
	}{
		{
			"Resource with no name",
			Resources{{"Pod", "v1"}},
			[]v1alpha1.APIResourceReference{{Kind: "Pod", Version: "v1"}},
			false,
		},
		{
			"Resource with name",
			Resources{{"Pod", "v1", "memcached-pod"}},
			[]v1alpha1.APIResourceReference{{Kind: "Pod", Version: "v1", Name: "memcached-pod"}},
			false,
		},
		{
			"Resource with string literal name",
			Resources{{"Pod", "v1", `"memcached-pod"`}},
			[]v1alpha1.APIResourceReference{{Kind: "Pod", Version: "v1", Name: `"memcached-pod"`}},
			false,
		},
		{
			"Two resources",
			Resources{{"Pod", "v1", "memcached-pod"}, {"Service", "v1"}},
			[]v1alpha1.APIResourceReference{
				{Kind: "Pod", Version: "v1", Name: "memcached-pod"},
				{Kind: "Service", Version: "v1"},
			},
			false,
		},
		{
			"Empty resource string without quotes",
			Resources{{""}},
			[]v1alpha1.APIResourceReference{},
			true,
		},
		{
			"Empty resource string with quotes",
			Resources{{`""`}},
			[]v1alpha1.APIResourceReference{},
			true,
		},
		{
			"Resource string with no version",
			Resources{{"Memcached"}},
			[]v1alpha1.APIResourceReference{},
			true,
		},
	}

	for _, c := range cases {
		output, err := c.input.toResourceReferences()
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
				t.Errorf("%s: expected %s, got %s", c.description, c.exp, output)
			}
		}
	}
}
