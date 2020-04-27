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

package alpha

import (
	"encoding/json"
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	"k8s.io/client-go/kubernetes"
)

// getTestResults fetches the test pod logs and converts it into
// ScorecardOutput format
func getTestResults(client kubernetes.Interface, tests []ScorecardTest) (output v1alpha2.ScorecardOutput) {
	output.Results = make([]v1alpha2.ScorecardTestResult, 0)
	for i := 0; i < len(tests); i++ {
		t := tests[i]
		p := t.TestPod
		logBytes, err := getPodLog(client, p)
		if err != nil {
			r := v1alpha2.ScorecardTestResult{}
			r.Name = t.Name
			r.Description = t.Description
			r.Errors = []string{fmt.Sprintf("Error getting pod log %s", err.Error())}
			output.Results = append(output.Results, r)
		} else {
			// marshal pod log into ScorecardTestResult
			var sc v1alpha2.ScorecardTestResult
			err := json.Unmarshal(logBytes, &sc)
			if err != nil {
				r := v1alpha2.ScorecardTestResult{}
				r.Name = t.Name
				r.Description = t.Description
				r.Errors = []string{fmt.Sprintf("Error unmarshalling test result %s", err.Error())}
				output.Results = append(output.Results, r)
			} else {
				output.Results = append(output.Results, sc)
			}
		}
	}
	return output
}

// ListTests lists the scorecard tests as configured that would be
// run based on user selection
func ListTests(o Options) (output v1alpha2.ScorecardOutput, err error) {
	tests := selectTests(o.Selector, o.Config.Tests)
	if len(tests) == 0 {
		fmt.Println("no tests selected")
		return output, err
	}

	output.Results = make([]v1alpha2.ScorecardTestResult, 0)

	for i := 0; i < len(tests); i++ {
		testResult := v1alpha2.ScorecardTestResult{}
		testResult.Name = tests[i].Name
		testResult.Labels = tests[i].Labels
		testResult.Description = tests[i].Description
		output.Results = append(output.Results, testResult)
	}

	return output, err
}
