// Copyright 2021 The Operator-SDK Authors
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

package client

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/kube"
)

func TestContainsResourcePolicyKeep(t *testing.T) {
	tests := []struct {
		input       map[string]string
		expectedVal bool
		expectedOut string
		name        string
	}{
		{
			input: map[string]string{
				kube.ResourcePolicyAnno: kube.KeepPolicy,
			},
			expectedVal: true,
			name:        "base case true",
		},
		{
			input: map[string]string{
				"not-" + kube.ResourcePolicyAnno: kube.KeepPolicy,
			},
			expectedVal: false,
			name:        "base case annotation false",
		},
		{
			input: map[string]string{
				kube.ResourcePolicyAnno: "not-" + kube.KeepPolicy,
			},
			expectedVal: false,
			name:        "base case value false",
		},
		{
			input: map[string]string{
				kube.ResourcePolicyAnno: strings.ToUpper(kube.KeepPolicy),
			},
			expectedVal: true,
			name:        "true with upper case",
		},
		{
			input: map[string]string{
				kube.ResourcePolicyAnno: " " + kube.KeepPolicy + "  ",
			},
			expectedVal: true,
			name:        "true with spaces",
		},
		{
			input: map[string]string{
				kube.ResourcePolicyAnno: " " + strings.ToUpper(kube.KeepPolicy) + "  ",
			},
			expectedVal: true,
			name:        "true with upper case and spaces",
		},
	}

	for _, test := range tests {
		assert.Equal(t, test.expectedVal, containsResourcePolicyKeep(test.input), test.name)
	}
}
