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
	"github.com/operator-framework/operator-registry/pkg/registry"
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

const (
	CustomTest1Name = "customtest1"
	CustomTest2Name = "customtest2"
)

// Define any operator specific custom tests here.
// CustomTest1 and CustomTest2 are example test functions. Relevant operator specific
// test logic is to be implemented in similarly.

func CustomTest1(bundle registry.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = CustomTest1Name
	r.Description = "Custom Test 1"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	return r
}

func CustomTest2(bundle registry.Bundle) scapiv1alpha2.ScorecardTestResult {
	r := scapiv1alpha2.ScorecardTestResult{}
	r.Name = CustomTest2Name
	r.Description = "Custom Test 2"
	r.State = scapiv1alpha2.PassState
	r.Errors = make([]string, 0)
	r.Suggestions = make([]string, 0)
	return r
}
