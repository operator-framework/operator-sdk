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
	"context"
	"encoding/json"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	v1 "k8s.io/api/core/v1"
)

// getTestResult fetches the test pod log and converts it into
// Test format
func (r PodTestRunner) getTestStatus(ctx context.Context, p *v1.Pod) (output *v1alpha3.TestStatus) {
	logBytes, err := getPodLog(ctx, r.Client, p)
	if err != nil {
		return convertErrorToStatus(err, string(logBytes))
	}
	// marshal pod log into TestResult
	err = json.Unmarshal(logBytes, &output)
	if err != nil {
		return convertErrorToStatus(err, string(logBytes))
	}
	return output
}

// List lists the scorecard tests as configured that would be
// run based on user selection
func (o Scorecard) List() v1alpha3.TestList {
	output := v1alpha3.NewTestList()
	for _, stage := range o.Config.Stages {
		tests := o.selectTests(stage)
		for _, test := range tests {
			item := v1alpha3.NewTest()
			item.Spec = test
			output.Items = append(output.Items, item)
		}
	}
	return output
}
