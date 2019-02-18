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
