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
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
)

// The scorecard test chainsaw binary processes the
// output from chainsaw converting chainsaw output into the
// scorecard v1alpha3.TestStatus json format.
//
// The chainsaw output is expected to be produced by chainsaw
// at /tmp/chainsaw-report.json.
func main() {
	jsonFile, err := os.Open("/tmp/chainsaw-report.json")
	if err != nil {
		printErrorStatus(fmt.Errorf("could not open chainsaw report %v", err))
		return
	}
	defer jsonFile.Close()
	var byteValue []byte
	byteValue, err = io.ReadAll(jsonFile)
	if err != nil {
		printErrorStatus(fmt.Errorf("could not read chainsaw report %v", err))
		return
	}
	var jsonReport TestsReport
	err = json.Unmarshal(byteValue, &jsonReport)
	if err != nil {
		printErrorStatus(fmt.Errorf("could not unmarshal chainsaw report %v", err))
		return
	}
	if len(jsonReport.Reports) == 0 {
		printErrorStatus(errors.New("no chainsaw test suite was found. chainsaw may not have run successfully"))
		return
	}
	s := getTestStatus(jsonReport.Reports)
	jsonOutput, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		printErrorStatus(fmt.Errorf("could not marshal scorecard output %v", err))
		return
	}
	fmt.Println(string(jsonOutput))
}

func getTestStatus(tc []*TestReport) (s v1alpha3.TestStatus) {
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
			r.Errors = []string{tc[i].Failure.Message}
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
	r.Log = getKuttlLogs()
	s.Results = append(s.Results, r)
	jsonOutput, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		log.Fatal(fmt.Errorf("could not marshal scorecard output %v", err))
	}
	fmt.Println(string(jsonOutput))
}

func getKuttlLogs() string {
	stderrFile, err := os.ReadFile("/tmp/chainsaw.stderr")
	if err != nil {
		return fmt.Sprintf("could not open chainsaw stderr file: %v", err)
	}
	stdoutFile, err := os.ReadFile("/tmp/chainsaw.stdout")
	if err != nil {
		return fmt.Sprintf("could not open chainsaw stdout file: %v", err)
	}
	return string(stderrFile) + string(stdoutFile)
}

// chainsaw report format
// the chainsaw structs below are copied from the kuttl master currently,
// in the future, these structs might be pulled into SDK as
// normal golang deps if necessary

type OperationType string

const (
	OperationTypeCreate  OperationType = "create"
	OperationTypeDelete  OperationType = "delete"
	OperationTypeApply   OperationType = "apply"
	OperationTypeAssert  OperationType = "assert"
	OperationTypeError   OperationType = "error"
	OperationTypeScript  OperationType = "script"
	OperationTypeSleep   OperationType = "sleep"
	OperationTypeCommand OperationType = "command"
)

// Failure represents details of a test failure.
type Failure struct {
	// Message provides a summary of the failure.
	Message string `json:"message" xml:"message,attr"`
}

// TestsReport encapsulates the entire report for a test suite.
type TestsReport struct {
	// Name of the test suite.
	Name string `json:"name" xml:"name,attr"`
	// TimeStamp marks when the test suite began execution.
	TimeStamp time.Time `json:"timestamp" xml:"timestamp,attr"`
	// Time indicates the total duration of the test suite.
	Time string `json:"time" xml:"time,attr"`
	// Test count the number of tests in the files/TestReports.
	Test int `json:"tests" xml:"tests,attr"`
	// Reports is an array of individual test reports within this suite.
	Reports []*TestReport `json:"testsuite" xml:"testsuite"`
	// Failures count the number of failed tests in the suite.
	Failures int `json:"failures" xml:"failures,attr"`
}

// TestReport represents a report for a single test.
type TestReport struct {
	// Name of the test.
	Name string `json:"name" xml:"name,attr"`
	// TimeStamp marks when the test began execution.
	TimeStamp time.Time `json:"timestamp" xml:"timestamp,attr"`
	// Time indicates the total duration of the test.
	Time string `json:"time" xml:"time,attr"`
	// Failure captures details if the test failed it should be nil otherwise.
	Failure *Failure `json:"failure,omitempty" xml:"failure,omitempty"`
	// Test count the number of tests in the suite/TestReport.
	Test int `json:"tests" xml:"tests,attr"`
	// Spec represents the specifications of the test.
	Steps []*TestSpecStepReport `json:"testcase,omitempty" xml:"testcase,omitempty"`
	// Concurrent indicates if the test runs concurrently with other tests.
	Concurrent bool `json:"concurrent,omitempty" xml:"concurrent,attr,omitempty"`
	// Namespace in which the test runs.
	Namespace string `json:"namespace,omitempty" xml:"namespace,attr,omitempty"`
	// Skip indicates if the test is skipped.
	Skip bool `json:"skip,omitempty" xml:"skip,attr,omitempty"`
	// SkipDelete indicates if resources are not deleted after test execution.
	SkipDelete bool `json:"skipDelete,omitempty" xml:"skipDelete,attr,omitempty"`
}

// TestSpecStepReport represents a report of a single step in a test.
type TestSpecStepReport struct {
	// Name of the test step.
	Name string `json:"name,omitempty" xml:"name,attr,omitempty"`
	// Results are the outcomes of operations performed in this step.
	Results []*OperationReport `json:"results,omitempty" xml:"results,omitempty"`
}

// OperationReport details the outcome of a single operation within a test step.
type OperationReport struct {
	// Name of the operation.
	Name string `json:"name" xml:"name,attr"`
	// TimeStamp marks when the operation began execution.
	TimeStamp time.Time `json:"timestamp" xml:"timestamp,attr"`
	// Time indicates the total duration of the operation.
	Time string `json:"time" xml:"time,attr"`
	// Result of the operation.
	Result string `json:"result" xml:"result,attr"`
	// Message provides additional information about the operation's outcome.
	Message string `json:"message,omitempty" xml:"message,omitempty"`
	// Type indicates the type of operation.
	OperationType OperationType `json:"operationType,omitempty" xml:"operationType,attr"`
}
