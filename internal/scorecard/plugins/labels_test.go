// Copyright 2019 The Operator-SDK Authors
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

package scplugins

import (
	"testing"

	"k8s.io/apimachinery/pkg/labels"
)

func TestBasicShortNames(t *testing.T) {
	specBlockExists := getStructShortName(CheckSpecTest{})
	statusBlockExists := getStructShortName(CheckStatusTest{})
	writeIntoCR := getStructShortName(WritingIntoCRsHasEffectTest{})

	cases := []struct {
		selectorValue string
		testsSelected int
		wantError     bool
	}{
		{"test=" + specBlockExists, 1, false},
		{"test in (" + specBlockExists + ")", 1, false},
		{"test in (" + specBlockExists + "," + writeIntoCR + ")", 2, false},
		{"test=" + statusBlockExists, 1, false},
		{"test=" + writeIntoCR, 1, false},
		{"testXwriteintocr", 0, false},
		{"test X writeintocr", 0, true},
	}

	for _, c := range cases {
		t.Run(c.selectorValue, func(t *testing.T) {
			var selector labels.Selector
			selector, err := labels.Parse(c.selectorValue)
			if err != nil && c.wantError {
				t.Logf("Wanted error and got error : %v", err)
				return
			} else if err != nil && !c.wantError {
				t.Errorf("Wanted result but got error: %v", err)
				return
			}

			basicTests := NewBasicTestSuite(BasicTestConfig{})
			basicTests.ApplySelector(selector)
			testsSelected := len(basicTests.Tests)
			if testsSelected != c.testsSelected {
				t.Errorf("Wanted testsSelected %d, got: %d", c.testsSelected, testsSelected)
			}
		})

	}
}

func TestOLMShortNames(t *testing.T) {
	bundleValidation := getStructShortName(BundleValidationTest{})
	crdValidationSection := getStructShortName(CRDsHaveValidationTest{})
	crdHasResources := getStructShortName(CRDsHaveResourcesTest{})
	specDescriptors := getStructShortName(SpecDescriptorsTest{})
	statusDescriptors := getStructShortName(StatusDescriptorsTest{})

	cases := []struct {
		selectorValue string
		testsSelected int
		wantError     bool
	}{
		{"test=" + bundleValidation, 1, false},
		{"test=" + crdValidationSection, 1, false},
		{"test=" + crdHasResources, 1, false},
		{"test=" + specDescriptors, 1, false},
		{"test=" + statusDescriptors, 1, false},
		{"testXstatusdescriptors", 0, false},
		{"test X statusdescriptors", 0, true},
	}

	for _, c := range cases {
		t.Run(c.selectorValue, func(t *testing.T) {
			var selector labels.Selector
			selector, err := labels.Parse(c.selectorValue)
			if err != nil && c.wantError {
				t.Logf("Wanted error and got error : %v", err)
				return
			}
			if err != nil && !c.wantError {
				t.Errorf("Wanted result but got error: %v", err)
				return
			}
			olmTests := NewOLMTestSuite(OLMTestConfig{})
			olmTests.ApplySelector(selector)
			testsSelected := len(olmTests.Tests)
			if testsSelected != c.testsSelected {
				t.Errorf("Wanted testsSelected %d, got: %d", c.testsSelected, testsSelected)
			}
		})

	}
}
