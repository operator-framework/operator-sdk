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
//	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	apimanifests "github.com/operator-framework/api/pkg/manifests"

	scorecard "github.com/operator-framework/operator-sdk/internal/scorecard/alpha"
//	scapiv1alpha3 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha3"
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
	//cfg, err := apimanifests.GetBundleFromDir(scorecard.PodBundleRoot)
	_, err := apimanifests.GetBundleFromDir(scorecard.PodBundleRoot)
	if err != nil {
		log.Fatal(err.Error())
	}

//	var result scapiv1alpha3.TestStatus

	switch entrypoint[0] {
	default:
		log.Printf("entrypoint called %s", entrypoint[0])
	}

	fmt.Printf("jeff sleeping inside the container binary")
	time.Sleep(time.Second * 10000)
/**
	prettyJSON, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		log.Fatal("failed to generate json", err)
	}
	fmt.Printf("%s\n", string(prettyJSON))
*/
	fmt.Printf("%s\n", "some kuttl results go here")

}

