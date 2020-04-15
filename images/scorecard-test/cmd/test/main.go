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

	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"

	"github.com/operator-framework/operator-sdk/internal/scorecard/alpha/tests"
)

// this is the scorecard test binary that ultimately executes the
// built-in scorecard tests (basic/olm).  The bundle that is under
// test is expected to be mounted so that tests can inspect the
// bundle contents as part of their test implementations.
// The actual test is to be run is named and that name is passed
// as an argument to this binary.  This argument mechanism allows
// this binary to run various tests all from within a single
// test image.

const (
	bundlePath = "/scorecard"
)

func main() {
	entrypoint := os.Args[1:]
	if len(entrypoint) == 0 {
		log.Fatal("test name argument is required")
	}

	cfg, err := tests.GetBundle(bundlePath)
	if err != nil {
		log.Fatal(err.Error())
	}

	var result scapiv1alpha2.ScorecardTestResult

	switch entrypoint[0] {
	case tests.OLMBundleValidationTest:
		result = tests.BundleValidationTest(cfg)
	case tests.OLMCRDsHaveValidationTest:
		result = tests.CRDsHaveValidationTest(cfg)
	case tests.OLMCRDsHaveResourcesTest:
		result = tests.CRDsHaveResourcesTest(cfg)
	case tests.OLMSpecDescriptorsTest:
		result = tests.SpecDescriptorsTest(cfg)
	case tests.OLMStatusDescriptorsTest:
		result = tests.StatusDescriptorsTest(cfg)
	case tests.BasicCheckStatusTest:
		result = tests.CheckStatusTest(cfg)
	case tests.BasicCheckSpecTest:
		result = tests.CheckSpecTest(cfg)
	default:
		log.Fatal("invalid test name argument passed")
		// TODO print out full list of test names to give a hint
		// to the end user on what the valid tests are
	}

	prettyJSON, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		log.Fatal("failed to generate json", err)
	}
	fmt.Printf("%s\n", string(prettyJSON))

}
