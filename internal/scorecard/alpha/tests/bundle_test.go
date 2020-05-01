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
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestBundlePath(t *testing.T) {
	cases := []struct {
		bundlePath string
		wantError  bool
	}{
		{"../testdata", false},
		{"/foo", true},
	}

	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {

			_, err := GetBundle(c.bundlePath)
			if err != nil && c.wantError {
				t.Logf("Wanted error and got error : %v", err)
				return
			} else if err != nil && !c.wantError {
				t.Errorf("Wanted result but got error: %v", err)
				return
			}

		})

	}
}
func TestBundleCRs(t *testing.T) {
	cases := []struct {
		bundlePath string
		crCount    int
		wantError  bool
	}{
		{"../testdata", 1, false},
	}

	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {

			bundle, err := GetBundle(c.bundlePath)
			if err != nil && c.wantError {
				t.Logf("Wanted error and got error : %v", err)
				return
			} else if err != nil && !c.wantError {
				t.Errorf("Wanted result but got error: %v", err)
				return
			}
			var crList []unstructured.Unstructured
			crList, err = GetCRs(*bundle)
			if err != nil {
				t.Error(err)
				return
			}
			if len(crList) != c.crCount {
				t.Errorf("Wanted %d CRs but got: %d", c.crCount, len(crList))
				return
			}

		})

	}
}

func TestBasicAndOLM(t *testing.T) {

	cases := []struct {
		bundlePath string
		state      scapiv1alpha2.State
		function   func(registry.Bundle) scapiv1alpha2.ScorecardTestResult
	}{
		{"../testdata", scapiv1alpha2.PassState, CheckSpecTest},
		{"../testdata/bundle", scapiv1alpha2.PassState, CRDsHaveValidationTest},
		{"../testdata", scapiv1alpha2.PassState, CRDsHaveResourcesTest},
		{"../testdata/bundle", scapiv1alpha2.PassState, SpecDescriptorsTest},
		{"../testdata", scapiv1alpha2.PassState, StatusDescriptorsTest},
	}

	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {

			bundle, err := GetBundle(c.bundlePath)
			if err != nil {
				t.Errorf("Error getting bundle %s", err.Error())
			}

			result := c.function(*bundle)
			if result.State != c.state {
				t.Errorf("%s result State %v expected", result.Name, c.state)
				return
			}
		})
	}
}

func TestOLMBundle(t *testing.T) {
	cases := []struct {
		bundlePath string
		state      scapiv1alpha2.State
	}{
		{"../testdata/bundle", scapiv1alpha2.PassState},
	}
	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {
			result := BundleValidationTest(c.bundlePath)
			if result.State != c.state {
				t.Errorf("%s result State %v expected", result.Name, c.state)
				return
			}
		})
	}
}

func TestCRDHaveValidation(t *testing.T) {

	cases := []struct {
		bundlePath string
		state      scapiv1alpha2.State
		function   func(registry.Bundle) scapiv1alpha2.ScorecardTestResult
	}{
		{"../testdata/crdvalidation/error_bundle", scapiv1alpha2.ErrorState, CRDsHaveValidationTest},
		{"../testdata/crdvalidation/invalid_status_bundle", scapiv1alpha2.FailState, CRDsHaveValidationTest},
		{"../testdata/crdvalidation/invalid_spec_bundle", scapiv1alpha2.FailState, CRDsHaveValidationTest},
		{"../testdata/crdvalidation/invalid_version_kind_check", scapiv1alpha2.PassState, CRDsHaveValidationTest},
	}

	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {
			bundle, err := GetBundle(c.bundlePath)
			if err != nil {
				t.Errorf("Error getting bundle %s", err.Error())
			}
			result := c.function(*bundle)
			if result.State != c.state {
				t.Errorf("%s result State %v expected", result.Name, c.state)
				return
			}
		})
	}
}
