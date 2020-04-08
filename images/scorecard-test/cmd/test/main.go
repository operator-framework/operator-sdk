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
	_, err := tests.GetConfig(bundlePath)
	if err != nil {
		// TODO produce a v1alpha2 Test Result error in the log
	}
	//	log.Printf("cfg %v\n", cfg)

}
