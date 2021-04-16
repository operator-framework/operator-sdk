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
	"flag"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-sdk/hack/generate/samples/internal/ansible"
	"github.com/operator-framework/operator-sdk/internal/testutils"
)

// This generate is used to run the e2e molecule tests
func main() {
	var (
		// binaryPath allow inform the binary that should be used.
		// By default it is operator-sdk
		binaryPath string

		// samplesRoot is the path provided to generate the molecule sample
		samplesRoot string

		// sample is the name of the mock was selected to be generated
		sample string
	)

	// testdata is the path where all samples are generate
	const testdata = "/testdata/"

	flag.StringVar(&binaryPath, "bin", testutils.BinaryName, "Binary path that should be used")
	flag.StringVar(&samplesRoot, "samples-root", "", "Path where molecule samples should be generated")
	flag.StringVar(&sample, "sample", "", "To generate only the selected option. Options: [advanced, memcached]")

	flag.Parse()

	// Make the binary path absolute if pathed, for reproducibility and debugging purposes.
	if dir, _ := filepath.Split(binaryPath); dir != "" {
		tmp, err := filepath.Abs(binaryPath)
		if err != nil {
			log.Fatalf("Failed to make binary path %q absolute: %v", binaryPath, err)
		}
		binaryPath = tmp
	}

	// If no path be provided then the Molecule sample will be create in the testdata/ansible dir
	// It can be helpful to check the mock data used in the e2e molecule tests as to develop this sample
	// By default this mock is ignored in the .gitignore
	if strings.TrimSpace(samplesRoot) == "" {
		currentPath, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		samplesRoot = filepath.Join(currentPath, testdata, "ansible")
	}

	log.Infof("creating Ansible Molecule Mock Samples under %s", samplesRoot)

	if sample == "" || sample == "memcached" {
		ansible.GenerateMoleculeSample(binaryPath, samplesRoot)
	}

	if sample == "" || sample == "advanced" {
		ansible.GenerateAdvancedMoleculeSample(binaryPath, samplesRoot)
	}
}
