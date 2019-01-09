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

	"github.com/operator-framework/operator-sdk/pkg/scaffold"
)

func TestConfig(t *testing.T) {
	crdsDir := filepath.Join(testDataDir, scaffold.CrdsDir)

	testConfig := &CSVConfig{
		CrdCrPaths: []string{crdsDir},
	}
	if err := testConfig.setFields(); err != nil {
		t.Errorf("set fields crd-cr paths dir only: (%v)", err)
	}
	if len(testConfig.CrdCrPaths) != 2 {
		t.Errorf("wanted 2 crd/cr files, got: %v", testConfig.CrdCrPaths)
	}

	testConfig = &CSVConfig{
		CrdCrPaths: []string{crdsDir, filepath.Join(crdsDir, "app_v1alpha1_app_crd.yaml")},
	}
	if err := testConfig.setFields(); err != nil {
		t.Errorf("set fields crd-cr paths dir file mix: (%v)", err)
	}
	want := []string{
		filepath.Join(crdsDir, "app_v1alpha1_app_cr.yaml"),
		filepath.Join(crdsDir, "app_v1alpha1_app_crd.yaml"),
	}
	if !reflect.DeepEqual(want, testConfig.CrdCrPaths) {
		t.Errorf("wanted crd/cr files %v, got %v", want, testConfig.CrdCrPaths)
	}
}
