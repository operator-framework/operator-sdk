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
	//bundlePath = "/scorecard" // mounted by the Pod
	bundlePath = "/home/jeffmc/projects/memcached-operator/deploy/olm-catalog/memcached-operator" // mounted by the Pod
)

func main() {
	log.Printf("scorecard-test starting\n")

	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) == 0 {
		// todo return an error since a test name is required
		log.Printf("test name argument is required")
		os.Exit(2)
	}

	cfg, err := tests.GetConfig(bundlePath)
	if err != nil {
		// TODO produce a v1alpha2 Test Result error in the log
	}
	log.Printf("cfg had %d CRs\n", len(cfg.CRs))

	var results []scapiv1alpha2.ScorecardTestResult

	log.Printf("entrypoint [%s]\n", argsWithoutProg[0])
	switch argsWithoutProg[0] {
	case tests.OLMBundleValidationTest:
		results = tests.BundleValidationTest(cfg)
	case tests.OLMCRDsHaveValidationTest:
		results = tests.CRDsHaveValidationTest(cfg)
	case tests.OLMCRDsHaveResourcesTest:
		results = tests.CRDsHaveResourcesTest(cfg)
	case tests.OLMSpecDescriptorsTest:
		results = tests.SpecDescriptorsTest(cfg)
	case tests.OLMStatusDescriptorsTest:
		results = tests.StatusDescriptorsTest(cfg)
	case tests.BasicCheckStatusTest:
		results = tests.CheckStatusTest(cfg)
	case tests.BasicCheckSpecTest:
		results = tests.CheckSpecTest(cfg)
	default:
		// todo error if an invalid test name is passed
		log.Printf("invalid test name argument passed\n")
		os.Exit(2)
	}

	log.Printf("%v\n", results)

}
