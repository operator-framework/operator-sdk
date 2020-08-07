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

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	"k8s.io/apimachinery/pkg/labels"
)

func TestEmptySelector(t *testing.T) {

	cases := []struct {
		selectorValue string
		testsSelected int
		config        v1alpha3.Configuration
		wantError     bool
	}{
		{"", 7, testConfig, false},
		{"suite in (kuttl)", 1, testConfig, false},
		{"test=basic-check-spec-test", 1, testConfig, false},
		{"testXwriteintocr", 0, testConfig, false},
		{"test X writeintocr", 0, testConfig, true},
	}

	for _, c := range cases {
		t.Run(c.selectorValue, func(t *testing.T) {
			o := Scorecard{}
			o.Config = c.config

			var err error
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

var testConfig = v1alpha3.Configuration{
	Stages: []v1alpha3.StageConfiguration{
		{
			Tests: []v1alpha3.TestConfiguration{
				{Image: "quay.io/someuser/customtest1:v0.0.1",
					Entrypoint: []string{
						"custom-test",
					},
					Labels: map[string]string{
						"suite": "custom",
						"test":  "customtest1",
					},
				},

				{Image: "quay.io/someuser/customtest2:v0.0.1",
					Entrypoint: []string{
						"custom-test",
					},
					Labels: map[string]string{
						"suite": "custom",
						"test":  "customtest2",
					},
				},

				{Image: "quay.io/redhat/basictests:v0.0.1",
					Entrypoint: []string{
						"scorecard-test",
						"basic-check-spec",
					},
					Labels: map[string]string{
						"suite": "basic",
						"test":  "basic-check-spec-test",
					},
				},

				{Image: "quay.io/redhat/basictests:v0.0.1",
					Entrypoint: []string{
						"scorecard-test",
						"basic-check-status",
					},
					Labels: map[string]string{
						"suite": "basic",
						"test":  "basic-check-status-test",
					},
				},

				{Image: "quay.io/redhat/olmtests:v0.0.1",
					Entrypoint: []string{
						"scorecard-test",
						"olm-bundle-validation",
					},
					Labels: map[string]string{
						"suite": "olm",
						"test":  "olm-bundle-validation-test",
					},
				},

				{Image: "quay.io/redhat/olmtests:v0.0.1",
					Entrypoint: []string{
						"scorecard-test",
						"olm-crds-have-validation",
					},
					Labels: map[string]string{
						"suite": "olm",
						"test":  "olm-crds-have-validation-test",
					},
				},
				{Image: "quay.io/redhat/kuttltests:v0.0.1",
					Entrypoint: []string{
						"kuttl-test",
						"olm-status-descriptors",
					},
					Labels: map[string]string{
						"suite": "kuttl",
					},
				},
			},
		},
	},
}
