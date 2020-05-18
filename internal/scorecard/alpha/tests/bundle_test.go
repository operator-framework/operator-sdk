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
	"path/filepath"
	"testing"

	"github.com/operator-framework/api/pkg/operators"
	"github.com/operator-framework/operator-registry/pkg/registry"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

var testBundle = filepath.Join("..", "testdata", "bundle")

func TestBundlePath(t *testing.T) {
	cases := []struct {
		bundlePath string
		wantError  bool
	}{
		{testBundle, false},
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
		{testBundle, 1, false},
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
		{testBundle, scapiv1alpha2.PassState, CheckSpecTest},
		{testBundle, scapiv1alpha2.PassState, CRDsHaveValidationTest},
		{testBundle, scapiv1alpha2.PassState, CRDsHaveResourcesTest},
		{testBundle, scapiv1alpha2.PassState, SpecDescriptorsTest},
		{testBundle, scapiv1alpha2.PassState, StatusDescriptorsTest},
	}

	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {

			bundle, err := GetBundle(c.bundlePath)
			if err != nil {
				t.Fatalf("Error getting bundle: %s", err.Error())
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
		{testBundle, scapiv1alpha2.PassState},
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

func TestScorecardFuncs(t *testing.T) {

	cases := []struct {
		name       string
		bundlePath string
		state      scapiv1alpha2.State
		function   func(registry.Bundle) scapiv1alpha2.ScorecardTestResult
	}{
		{
			// test StatusDescriptorsTest()
			name:       "Should error when CR has format errors",
			bundlePath: "../testdata/statusdescriptor/error_bundle",
			state:      scapiv1alpha2.ErrorState,
			function:   StatusDescriptorsTest,
		},

		{
			// test StatusDescriptorsTest()
			name:       "Should fail when CR has missing status from CSV",
			bundlePath: "../testdata/statusdescriptor/invalid_status_bundle",
			state:      scapiv1alpha2.FailState,
			function:   StatusDescriptorsTest,
		},
		{
			// test StatusDescriptorsTest()
			// This test checks for spec.customresourcedefinitions.owned presence, and fails
			// when missing from CSV.
			name:       "Should fail when owned CRD is missing from CSV",
			bundlePath: "../testdata/statusdescriptor/no_crd_bundle",
			state:      scapiv1alpha2.FailState,
			function:   StatusDescriptorsTest,
		},

		{
			// test StatusDescriptorsTest()
			name:       "Should fail when statusDescriptor is missing from CSV",
			bundlePath: "../testdata/statusdescriptor/no_statusdesc_bundle",
			state:      scapiv1alpha2.FailState,
			function:   StatusDescriptorsTest,
		},
		{
			// test CRDsHaveValidationTest()
			name:       "Should fail when CR has spec field missing",
			bundlePath: "../testdata/crdvalidation/invalid_spec_bundle",
			state:      scapiv1alpha2.FailState,
			function:   CRDsHaveValidationTest,
		},

		{
			// test CRDsHaveValidationTest()
			// This test should skip and pass when version/kind does not match for CR with CRD.
			name:       "Should pass when CR has no matching version/kind",
			bundlePath: "../testdata/crdvalidation/invalid_version_kind_check",
			state:      scapiv1alpha2.PassState,
			function:   CRDsHaveValidationTest,
		},

		{
			// test CRDsHaveValidationTest()
			name:       "This test should error when CR has format issues",
			bundlePath: "../testdata/crdvalidation/error_bundle",
			state:      scapiv1alpha2.ErrorState,
			function:   CRDsHaveValidationTest,
		},

		{
			// test CRDsHaveValidationTest()
			name:       "Should fail when CR has status field missing",
			bundlePath: "../testdata/crdvalidation/invalid_status_bundle",
			state:      scapiv1alpha2.FailState,
			function:   CRDsHaveValidationTest,
		},
	}

	for _, c := range cases {
		t.Run(c.bundlePath, func(t *testing.T) {

			bundle, err := GetBundle(c.bundlePath)
			if err != nil {
				t.Errorf("Error when getting bundle %s", err.Error())
			}
			result := c.function(*bundle)
			if result.State != c.state {
				t.Errorf("%s is the result State, %v expected", result.Name, c.state)
				return
			}
		})
	}
}

func TestCheckForResources(t *testing.T) {
	crdWithResources := operators.CustomResourceDefinitions{
		Owned: []operators.CRDDescription{
			operators.CRDDescription{
				Name:              "Test",
				Version:           "v1",
				Kind:              "Test",
				StatusDescriptors: make([]operators.StatusDescriptor, 0),
				Resources: []operators.APIResourceReference{
					operators.APIResourceReference{
						Name:    "operator",
						Kind:    "Test",
						Version: "v1",
					},
				},
			},
		},
	}

	crdWithoutResources := operators.CustomResourceDefinitions{
		Owned: []operators.CRDDescription{
			operators.CRDDescription{
				Name:              "Test",
				Version:           "v1",
				Kind:              "Test",
				StatusDescriptors: make([]operators.StatusDescriptor, 0),
				Resources:         make([]operators.APIResourceReference, 0),
			},
		},
	}

	tests := []struct {
		name string
		args operators.CustomResourceDefinitions
		want scapiv1alpha2.State
	}{
		{
			name: "Should pass when CSV has Owned CRD's with resources",
			args: crdWithResources,
			want: scapiv1alpha2.PassState,
		},
		{
			name: "Should fail when CSV does not have Owned CRD's with resources",
			args: crdWithoutResources,
			want: scapiv1alpha2.FailState,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := scapiv1alpha2.ScorecardTestResult{
				Name:   "Check if CRDs have resources",
				State:  scapiv1alpha2.PassState,
				Errors: make([]string, 0),
			}
			if res = CheckResources(tt.args, res); res.State != tt.want {
				t.Errorf("%s result State %v expected but obtained %v",
					res.Name, tt.want, res.State)
			}
		})
	}
}
