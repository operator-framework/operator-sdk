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
		// binaryName allow inform the binary that should be used.
		// By default it is operator-sdk
		binaryName string

		// path is the path provided to generate the molecule sample
		path string
	)

	// testdata is the path where all samples are generate
	const testdata = "/testdata/"

	flag.StringVar(&binaryName, "bin", testutils.BinaryName, "Binary path that should be used")
	flag.StringVar(&path, "path", "", "Path where the molecule should be called")

	flag.Parse()

	// If no path be provided then the Molecule sample will be create in the testdata/ansible dir
	// It can be helpful to check the mock data used in the e2e molecule tests as to develop this sample
	// By default this mock is ignored in the .gitignore
	if strings.TrimSpace(path) == "" {
		currentPath, err := os.Getwd()
		if err != nil {
			log.Error(err)
			os.Exit(1)
		}
		path = filepath.Join(currentPath, testdata, "ansible")
	}

	log.Infof("creating Ansible Molecule Mock Sample")
	log.Infof("using the path: (%v)", path)
	ansible.GenerateMoleculeAnsibleSample(path)
}
