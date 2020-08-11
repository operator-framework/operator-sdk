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
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
)

// The scorecard test kuttl binary processes the
// output from kuttl converting kuttl output into the
// scorecard v1alpha3.TestStatus json format.
//
// The kuttl output is expected to be produced by kubectl-kuttl
// at /tmp/kuttl-test.json.
func main() {

	jsonFile, err := os.Open("/tmp/kuttl-test.json")
	if err != nil {
		printErrorStatus(fmt.Errorf("could not open kuttl report %v", err))
		return
	}
	defer jsonFile.Close()

	var byteValue []byte
	byteValue, err = ioutil.ReadAll(jsonFile)
	if err != nil {
		printErrorStatus(fmt.Errorf("could not read kuttl report %v", err))
		return
	}

	var jsonReport Testsuites
	err = json.Unmarshal(byteValue, &jsonReport)
	if err != nil {
		printErrorStatus(fmt.Errorf("could not unmarshal kuttl report %v", err))
		return
	}

	var suite *Testsuite
	if len(jsonReport.Testsuite) == 0 {
		printErrorStatus(errors.New("empty kuttl test suite was found"))
		return
	}

	suite = jsonReport.Testsuite[0]

	s := getTestStatus(suite.Testcase)

	jsonOutput, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		printErrorStatus(fmt.Errorf("could not marshal scoreard output %v", err))
		return
	}
	fmt.Println(string(jsonOutput))
}

func getTestStatus(tc []*Testcase) (s v1alpha3.TestStatus) {

	// report the kuttl logs when kuttl tests can not be run
	// (e.g. RBAC is not sufficient)
	if len(tc) == 0 {
		r := v1alpha3.TestResult{}
		r.Log = getKuttlLogs()
		s.Results = append(s.Results, r)
		return s
	}

	for i := 0; i < len(tc); i++ {
		r := v1alpha3.TestResult{}
		r.Name = tc[i].Name
		r.State = v1alpha3.PassState
		if tc[i].Failure != nil {
			r.State = v1alpha3.FailState
			r.Errors = []string{tc[i].Failure.Text}
		}
		s.Results = append(s.Results, r)
	}

	return s
}

func printErrorStatus(err error) {
	s := v1alpha3.TestStatus{}
	r := v1alpha3.TestResult{}
	r.State = v1alpha3.FailState
	r.Errors = []string{err.Error()}
	s.Results = append(s.Results, r)
	jsonOutput, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		log.Fatal(fmt.Errorf("could not marshal scorecard output %v", err))
	}
	fmt.Println(string(jsonOutput))
}

// kuttl report format
// the kuttl structs below are copied from the kuttl master currently,
// in the future, these structs might be pulled into SDK as
// normal golang deps if necessary

// Property are name/value pairs which can be provided in the report for things such as kuttl.version
type Property struct {
	Name  string `xml:"name,attr" json:"name"`
	Value string `xml:"value,attr" json:"value"`
}

// Properties defines the collection of properties
type Properties struct {
	Property []Property `xml:"property" json:"property,omitempty"`
}

// Failure defines a test failure
type Failure struct {
	// Text provides detailed information regarding failure.  It supports multi-line output.
	Text string `xml:",chardata" json:"text,omitempty"`
	// Message provides the summary of the failure
	Message string `xml:"message,attr" json:"message"`
	Type    string `xml:"type,attr" json:"type,omitempty"`
}

// Testcase is the finest grain level of reporting, it is the kuttl test (which contains steps)
type Testcase struct {
	// Classname is a junit thing, for kuttl it is the testsuite name
	Classname string `xml:"classname,attr" json:"classname"`
	// Name is the name of the test (folder of test if not redefined by the TestStep)
	Name string `xml:"name,attr" json:"name"`
	// Time is the elapsed time of the test (and all of it's steps)
	Time string `xml:"time,attr" json:"time"`
	// Assertions is the number of asserts and errors defined in the test
	Assertions int `xml:"assertions,attr" json:"assertions,omitempty"`
	// Failure defines a failure in this testcase
	Failure *Failure `xml:"failure" json:"failure,omitempty"`
}

// TestSuite is a collection of Testcase and is a summary of those details
type Testsuite struct {
	// Tests is the number of Testcases in the collection
	Tests int `xml:"tests,attr" json:"tests"`
	// Failures is the summary number of all failure in the collection testcases
	Failures int `xml:"failures,attr" json:"failures"`
	// Time is the duration of time for this Testsuite, this is tricky as tests run concurrently.
	// This is the elapse time between the start of the testsuite and the end of the latest testcase in the collection.
	Time string `xml:"time,attr" json:"time"`
	// Name is the kuttl test name
	Name string `xml:"name,attr" json:"name"`
	// Properties which are specific to this suite
	Properties *Properties `xml:"properties" json:"properties,omitempty"`
	// Testcase is a collection of test cases
	Testcase []*Testcase `xml:"testcase" json:"testcase,omitempty"`
}

// Testsuites is a collection of Testsuite and defines the rollup summary of all stats.
type Testsuites struct {
	// XMLName is required to refine the name (or case of the name)
	//in the root xml element.  Otherwise it adds no value and is ignored for json output.
	XMLName xml.Name `json:"-"`
	// Name is the name of the full set of tests which is possible to set in kuttl but is rarely used :)
	Name string `xml:"name,attr" json:"name"`
	// Tests is a summary value of the total number of tests for all testsuites
	Tests int `xml:"tests,attr" json:"tests"`
	// Failures is a summary value of the total number of failures for all testsuites
	Failures int `xml:"failures,attr" json:"failures"`
	// Time is the elapsed time of the entire suite of tests
	Time string `xml:"time,attr" json:"time"`
	// Properties which are for the entire set of tests
	Properties *Properties `xml:"properties" json:"properties,omitempty"`
	// Testsuite is a collection of test suites
	Testsuite []*Testsuite `xml:"testsuite" json:"testsuite,omitempty"`
}

func getKuttlLogs() string {
	stderrFile, err := ioutil.ReadFile("/tmp/kuttl.stderr")
	if err != nil {
		printErrorStatus(fmt.Errorf("could not open kuttl stderr file %v", err))
		return err.Error()
	}

	stdoutFile, err := ioutil.ReadFile("/tmp/kuttl.stdout")
	if err != nil {
		printErrorStatus(fmt.Errorf("could not open kuttl stdout file %v", err))
		return err.Error()
	}

	return string(stderrFile) + string(stdoutFile)
}
