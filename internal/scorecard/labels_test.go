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

package scorecard

import (
	"testing"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/labels"
)

func TestEmptySelector(t *testing.T) {

	cases := []struct {
		selectorValue string
		testsSelected int
		wantError     bool
	}{
		{"", 7, false},
		{"suite in (kuttl)", 1, false},
		{"test=basic-check-spec-test", 1, false},
		{"testXwriteintocr", 0, false},
		{"test X writeintocr", 0, true},
	}

	for _, c := range cases {
		t.Run(c.selectorValue, func(t *testing.T) {
			o := Scorecard{}

			err := yaml.Unmarshal([]byte(testConfig), &o.Config)
			if err != nil {
				t.Log(err)
				return
			}

			o.Selector, err = labels.Parse(c.selectorValue)
			if err == nil && c.wantError {
				t.Fatalf("Wanted error but got no error")
			} else if err != nil {
				if !c.wantError {
					t.Fatalf("Wanted result but got error: %v", err)
				}
				return
			}

			tests := o.selectTests(o.Config.Stages[0])
			testsSelected := len(tests)
			if testsSelected != c.testsSelected {
				t.Errorf("Wanted testsSelected %d, got: %d", c.testsSelected, testsSelected)
			}
		})

	}
}

const testConfig = `stages:
- tests:
  - image: quay.io/someuser/customtest1:v0.0.1
    entrypoint: 
    - custom-test
    labels:
      suite: custom
      test: customtest1
  - image: quay.io/someuser/customtest2:v0.0.1
    entrypoint: 
    - custom-test
    labels:
      suite: custom
      test: customtest2
  - image: quay.io/redhat/basictests:v0.0.1
    entrypoint: 
    - scorecard-test
    - basic-check-spec
    labels:
      suite: basic
      test: basic-check-spec-test
  - image: quay.io/redhat/basictests:v0.0.1
    entrypoint: 
    - scorecard-test
    - basic-check-status
    labels:
      suite: basic
      test: basic-check-status-test
  - image: quay.io/redhat/olmtests:v0.0.1
    entrypoint: 
    - scorecard-test
    - olm-bundle-validation
    labels:
      suite: olm
      test: olm-bundle-validation-test
  - image: quay.io/redhat/olmtests:v0.0.1
    entrypoint: 
    - scorecard-test
    - olm-crds-have-validation
    labels:
      suite: olm
      test: olm-crds-have-validation-test
  - image: quay.io/redhat/kuttltests:v0.0.1
    labels:
      suite: kuttl
    entrypoint:
    - kuttl-test
    - olm-status-descriptors
`
