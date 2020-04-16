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
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

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
	bundleZip = "/scorecard/bundle.zip"
)

func main() {
	entrypoint := os.Args[1:]
	if len(entrypoint) == 0 {
		log.Fatal("test name argument is required")
	}

	// Create tmp directory for the unzipped bundle
	tmpDir, err := ioutil.TempDir("/tmp", "scorecard-bundle")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(tmpDir)

	// TODO remove this log
	log.Printf("directory %s\n", tmpDir)

	// Unzip the bundle
	_, err = Unzip(bundleZip, tmpDir)
	if err != nil {
		log.Fatalf("error unzipping bundle %s", err.Error())
	}

	cfg, err := tests.GetBundle(tmpDir)
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

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
