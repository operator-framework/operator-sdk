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

package ansible

import (
	"os"
	"strconv"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/stretchr/testify/assert"
)

// TODO: add a test for the Run method

func TestMaxWorkers(t *testing.T) {
	testCases := []struct {
		name      string
		gvk       schema.GroupVersionKind
		defvalue  int
		expected  int
		setenvvar bool
		envvar    string
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
			envvar:    "WORKER_MEMCACHESERVICE_CACHE_EXAMPLE_COM",
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
			envvar:    "WORKER_MEMCACHESERVICE_CACHE_EXAMPLE_COM",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Unsetenv(tc.envvar)
			if tc.setenvvar {
				os.Setenv(tc.envvar, strconv.Itoa(tc.expected))
			}
			assert.Equal(t, tc.expected, getMaxWorkers(tc.gvk, tc.defvalue))
		})
	}
}
