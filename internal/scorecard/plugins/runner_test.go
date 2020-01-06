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

package scplugins

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

const (
	cr0 = `
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: example-memcached
spec:
  size: 3
status:
  nodes:
`
	cr1 = `
apiVersion: cache.example.com/v1alpha2
kind: Memcached
metadata:
  name: example-memcached
spec:
  size: 3
status:
  nodes:
`
	badCR = `
 apiVersion: cache.example.com/v1alpha2
 k ind: Memcached
ame tadata:
   name: example-memcached
sp ec:
	  size: 3
s  tatus :
 no  des:
`
)

func TestDuplicateCR(t *testing.T) {

	duplicateLogMessage := "Duplicate gvks in CR list detected"
	log = logrus.New()
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)

	var err error
	var cr0File *os.File
	var cr1File *os.File
	var badcrFile *os.File

	if cr0File, err = createCRFile(cr0); err != nil {
		t.Fatal(err)
		return
	}
	if cr1File, err = createCRFile(cr1); err != nil {
		t.Fatal(err)
		return
	}
	if badcrFile, err = createCRFile(badCR); err != nil {
		t.Fatal(err)
		return
	}
	defer os.Remove(cr0File.Name())
	defer os.Remove(cr1File.Name())
	defer os.Remove(badcrFile.Name())

	cases := []struct {
		testDescription string
		crA             string
		crB             string
		wantError       bool
		wantLog         bool
	}{
		{"duplicate CRs", cr0File.Name(), cr0File.Name(), false, true},
		{"unique CRs", cr0File.Name(), cr1File.Name(), false, false},
		{"bad CRs", badcrFile.Name(), cr1File.Name(), true, false},
	}

	for _, c := range cases {
		t.Run(c.testDescription, func(t *testing.T) {
			crs := []string{c.crA, c.crB}
			err := duplicateCRCheck(crs)
			if err == nil && !c.wantError && c.wantLog {
				logContents := logBuffer.String()
				if strings.Contains(logContents, duplicateLogMessage) {
					t.Logf("Wanted log and got log : %s", logBuffer.String())
				} else {
					t.Errorf("Wanted log to contain %s but not found\n", duplicateLogMessage)
					return
				}
			}

			if err != nil && c.wantError {
				t.Logf("Wanted error and got error : %v", err)
				return
			}

			if err != nil && !c.wantError {
				t.Errorf("Wanted result but got error: %v", err)
				return
			}

		})

	}
}

func createCRFile(contents string) (*os.File, error) {
	tmpFile0, err := ioutil.TempFile(os.TempDir(), "runnerTest-")
	if err != nil {
		return nil, err
	}

	fmt.Println("Created file: " + tmpFile0.Name())

	if _, err = tmpFile0.Write([]byte(contents)); err != nil {
		return nil, err
	}

	if err := tmpFile0.Close(); err != nil {
		return nil, err
	}
	return tmpFile0, nil
}
