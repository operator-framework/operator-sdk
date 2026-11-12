// Copyright 2026 The Operator-SDK Authors
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

package run

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/operator-framework/operator-sdk/internal/util/k8sutil"
)

func TestConfigureWatchNamespaces(t *testing.T) {
	tests := []struct {
		name           string
		watchNamespace string
		want           map[string]cache.Config
	}{
		{
			// A label selector on the all-namespaces config would take
			// precedence over the chart-label DefaultLabelSelector and cache
			// every object of each dependent kind in the cluster.
			name:           "Should leave the label selector unset when watching all namespaces",
			watchNamespace: "",
			want:           map[string]cache.Config{metav1.NamespaceAll: {}},
		},
		{
			name:           "Should configure each listed namespace",
			watchNamespace: "ns1, ns2",
			want:           map[string]cache.Config{"ns1": {}, "ns2": {}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(k8sutil.WatchNamespaceEnvVar, tc.watchNamespace)
			options := manager.Options{}

			configureWatchNamespaces(&options, logr.Discard())

			assert.Equal(t, tc.want, options.Cache.DefaultNamespaces)
		})
	}
}
