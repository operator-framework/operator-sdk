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

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/operator-framework/operator-sdk/internal/scorecard/alpha/tests"
	"github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

// this is the scorecard test binary that ultimately executes the
// built-in scorecard tests (basic/olm).  The bundle that is under
// test is expected to be mounted so that tests can inspect the
// bundle contents as part of their test implementations.
// The actual test is to be run is named and that name is passed
// as an argument to this binary.  This argument mechanism allows
// this binary to run various tests all from within a single
// test image.

func main() {
	entrypoint := os.Args[1:]
	if len(entrypoint) == 0 {
		log.Fatal("test name argument is required")
	}

	// This is the path that holds the untar'd bundle contents
	tmpDir := "/bundle"

	cfg, err := tests.GetBundle(tmpDir)
	if err != nil {
		log.Fatal(err.Error())
	}

	var result scapiv1alpha2.ScorecardTestResult

	switch entrypoint[0] {
	case tests.OLMBundleValidationTest:
		result = tests.BundleValidationTest(*cfg)
	case tests.OLMCRDsHaveValidationTest:
		result = tests.CRDsHaveValidationTest(*cfg)
	case tests.OLMCRDsHaveResourcesTest:
		result = tests.CRDsHaveResourcesTest(*cfg)
	case tests.OLMSpecDescriptorsTest:
		result = tests.SpecDescriptorsTest(*cfg)
	case tests.OLMStatusDescriptorsTest:
		result = tests.StatusDescriptorsTest(*cfg)
	case tests.BasicCheckSpecTest:
		result = tests.CheckSpecTest(*cfg)
	default:
		result = printValidTests()
	}

	prettyJSON, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		log.Fatal("failed to generate json", err)
	}
	fmt.Printf("%s\n", string(prettyJSON))

}

// printValidTests will print out full list of test names to give a hint to the end user on what the valid tests are
func printValidTests() (result v1alpha2.ScorecardTestResult) {
	result.State = scapiv1alpha2.FailState
	result.Errors = make([]string, 0)
	result.Suggestions = make([]string, 0)

	str := fmt.Sprintf("Valid tests for this image include: %s, %s, %s, %s, %s, %s",
		tests.OLMBundleValidationTest,
		tests.OLMCRDsHaveValidationTest,
		tests.OLMCRDsHaveResourcesTest,
		tests.OLMSpecDescriptorsTest,
		tests.OLMStatusDescriptorsTest,
		tests.BasicCheckSpecTest)
	result.Errors = append(result.Errors, str)
	return result
}
