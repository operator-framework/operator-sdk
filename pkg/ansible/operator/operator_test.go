// Copyright 2018 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package operator

import (
	"os"
	"strconv"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/stretchr/testify/assert"
)

// TODO: add a test for the Run method

func TestFormatEnvVar(t *testing.T) {
	testCases := []struct {
		name     string
		kind     string
		group    string
		expected string
	}{
		{
			name:     "easy path",
			kind:     "FooCluster",
			group:    "cache.example.com",
			expected: "WORKER_FOOCLUSTER_CACHE_EXAMPLE_COM",
		},
		{
			name:     "missing kind",
			kind:     "",
			group:    "cache.example.com",
			expected: "WORKER__CACHE_EXAMPLE_COM",
		},
		{
			name:     "missing group",
			kind:     "FooCluster",
			group:    "",
			expected: "WORKER_FOOCLUSTER_",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, formatEnvVar(tc.kind, tc.group))
		})
	}
}

func TestMaxWorkers(t *testing.T) {
	testCases := []struct {
		name      string
		gvk       schema.GroupVersionKind
		defvalue  int
		expected  int
		setenvvar bool
	}{
		{
			name: "no env, use default value",
			gvk: schema.GroupVersionKind{
				Group:   "cache.example.com",
				Version: "v1alpha1",
				Kind:    "MemCacheService",
			},
			defvalue:  1,
			expected:  1,
			setenvvar: false,
		},
		{
			name: "env set to 3, expect 3",
			gvk: schema.GroupVersionKind{
				Group:   "cache.example.com",
				Version: "v1alpha1",
				Kind:    "MemCacheService",
			},
			defvalue:  1,
			expected:  3,
			setenvvar: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Unsetenv(formatEnvVar(tc.gvk.Kind, tc.gvk.Group))
			if tc.setenvvar {
				os.Setenv(formatEnvVar(tc.gvk.Kind, tc.gvk.Group), strconv.Itoa(tc.expected))
			}
			assert.Equal(t, tc.expected, getMaxWorkers(tc.gvk, tc.defvalue))
		})
	}
}
