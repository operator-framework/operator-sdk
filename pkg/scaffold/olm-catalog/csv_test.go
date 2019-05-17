// Copyright 2018 The Operator-SDK Authors
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

package catalog

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/operator-framework/operator-sdk/internal/pkg/scaffold"
)

var testDataDir = filepath.Join("..", "..", "..", "test", "test-framework")

func TestConfig(t *testing.T) {
	crdsDir := filepath.Join(testDataDir, scaffold.CRDsDir)
	c := GenCSVCmd{
		CRDCRPaths: []string{
			filepath.Join(crdsDir, "cache_v1alpha1_memcached_crd.yaml"),
		},
	}
	if err := c.expandPaths(); err != nil {
		t.Errorf("File-only crd-cr paths: (%v)", err)
	}
	if len(c.CRDCRPaths) != 1 {
		t.Errorf("Wanted 1 CRD/CR files, got: %+q", c.CRDCRPaths)
	}

	c.CRDCRPaths = []string{crdsDir, filepath.Join(crdsDir, "doesntexist_v1alpha1_app_crd.yaml")}
	if err := c.expandPaths(); err != nil {
		t.Errorf("Existing dir and a non-existent CRD crd-cr paths: (%v)", err)
	}
	want := []string{
		filepath.Join(crdsDir, "cache_v1alpha1_memcached_cr.yaml"),
		filepath.Join(crdsDir, "cache_v1alpha1_memcached_crd.yaml"),
		filepath.Join(crdsDir, "cache_v1alpha1_memcachedrs_cr.yaml"),
		filepath.Join(crdsDir, "cache_v1alpha1_memcachedrs_crd.yaml"),
	}
	if !reflect.DeepEqual(want, c.CRDCRPaths) {
		t.Errorf("Wanted crd/cr files %v, got %+q", want, c.CRDCRPaths)
	}
}
