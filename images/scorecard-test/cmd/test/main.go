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

	apimanifests "github.com/operator-framework/api/pkg/manifests"

	registryutil "github.com/operator-framework/operator-sdk/internal/registry"
	"github.com/operator-framework/operator-sdk/internal/scorecard"
	"github.com/operator-framework/operator-sdk/internal/scorecard/tests"
	scapiv1alpha3 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha3"
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

	// Read the pod's untar'd bundle from a well-known path.
	bundle, err := apimanifests.GetBundleFromDir(scorecard.PodBundleRoot)
	if err != nil {
		log.Fatal(err.Error())
	}

	metadata, _, err := registryutil.FindBundleMetadata(scorecard.PodBundleRoot)
	if err != nil {
		log.Fatal(err.Error())
	}

	var result scapiv1alpha3.TestStatus

	switch entrypoint[0] {
	case tests.OLMBundleValidationTest:
		result = tests.BundleValidationTest(scorecard.PodBundleRoot, metadata)
	case tests.OLMCRDsHaveValidationTest:
		result = tests.CRDsHaveValidationTest(bundle)
	case tests.OLMCRDsHaveResourcesTest:
		result = tests.CRDsHaveResourcesTest(bundle)
	case tests.OLMSpecDescriptorsTest:
		result = tests.SpecDescriptorsTest(bundle)
	case tests.OLMStatusDescriptorsTest:
		result = tests.StatusDescriptorsTest(bundle)
	case tests.BasicCheckSpecTest:
		result = tests.CheckSpecTest(bundle)
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
func printValidTests() scapiv1alpha3.TestStatus {
	result := scapiv1alpha3.TestResult{}
	result.State = scapiv1alpha3.FailState
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
	return scapiv1alpha3.TestStatus{
		Results: []scapiv1alpha3.TestResult{result},
	}
}
