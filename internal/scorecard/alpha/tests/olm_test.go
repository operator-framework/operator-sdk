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

package tests

import (
	"testing"

	"github.com/operator-framework/operator-registry/pkg/registry"

	"github.com/operator-framework/operator-sdk/internal/scorecard/alpha/scorecardutil"
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

func TestOLM(t *testing.T) {

	cases := []struct {
		name       string
		bundlePath string
		state      scapiv1alpha2.State
		function   func(registry.Bundle) scapiv1alpha2.ScorecardTestResult
	}{
		{OLMBundleValidationTest, "../testdata/bundle", scapiv1alpha2.PassState, BundleValidationTest},
		{OLMCRDsHaveValidationTest, "../testdata/bundle", scapiv1alpha2.PassState, CRDsHaveValidationTest},
		{OLMCRDsHaveResourcesTest, "../testdata/bundle", scapiv1alpha2.PassState, CRDsHaveResourcesTest},
		{OLMSpecDescriptorsTest, "../testdata/bundle", scapiv1alpha2.PassState, SpecDescriptorsTest},
		{OLMStatusDescriptorsTest, "../testdata/bundle", scapiv1alpha2.PassState, StatusDescriptorsTest},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bundle, err := scorecardutil.LoadBundleDirectory(c.bundlePath)
			if err != nil {
				t.Fatalf("Error getting bundle %s", err.Error())
			}

			result := c.function(*bundle)
			if result.State != scapiv1alpha2.PassState {
				t.Errorf("%s result State %v expected", result.Name, scapiv1alpha2.PassState)
			}
		})
	}
}
