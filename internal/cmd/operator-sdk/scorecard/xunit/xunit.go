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

package xunitapi

import (
	"encoding/xml"
	"time"
)

// NewTestSuites returns a new XUnit result from the given test suites.
func NewTestSuites(name string, testSuites []TestSuite) TestSuites {
	return TestSuites{
		Name:       name,
		TestSuites: testSuites,
	}
}

// TestSuites is the top level object for amassing Xunit test results
type TestSuites struct {
	XMLName    xml.Name    `xml:"testsuites"` // Component name: <testsuites>
	Name       string      `xml:"name,attr"`
	TestSuites []TestSuite `xml:"testsuite"`
}

// Preperty is a named property that will be formatted as an XML tag.
type Property struct {
	Name  string      `xml:"name,attr"`
	Value interface{} `xml:"value,attr"`
}

// TestSuite contains for details about a test beyond the final status
type TestSuite struct {
	Name       string `xml:"name,attr"`
	Properties struct {
		Properties []Property `xml:"property"`
	} `xml:"properties,omitempty"`
	TestCases []TestCase `xml:"testcase,omitempty"`
	Tests     int        `xml:"tests,attr"`
	Skipped   int        `xml:"skipped,attr"`
	Failures  int        `xml:"failures,attr"`
	Errors    int        `xml:"errors,attr"`
}

// NewSuite creates a new test suite with the given name.
func NewSuite(name string) TestSuite {
	return TestSuite{Name: name}
}

// AddProperty adds the property key/value to the test suite.
func (ts *TestSuite) AddProperty(name, value string) {
	ts.Properties.Properties = append(ts.Properties.Properties, Property{Name: name, Value: value})
}

// AddSuccess adds a passing test case to the suite.
func (ts *TestSuite) AddSuccess(name string, time time.Time, logs string) {
	ts.addTest(name, time, logs, nil)
}

// AddFailure adds a failed test case to the suite.
func (ts *TestSuite) AddFailure(name string, time time.Time, logs, msg string) {
	ts.Failures++
	ts.addTest(name, time, logs, &Result{
		XMLName: xml.Name{Local: "failure"},
		Type:    "failure",
		Message: msg,
	})
}

// AddError adds an errored test case to the suite.
func (ts *TestSuite) AddError(name string, time time.Time, logs, msg string) {
	ts.Errors++
	ts.addTest(name, time, logs, &Result{
		XMLName: xml.Name{Local: "error"},
		Type:    "error",
		Message: msg,
	})
}

func (ts *TestSuite) addTest(name string, time time.Time, logs string, result *Result) {
	ts.Tests++
	ts.TestCases = append(ts.TestCases, TestCase{
		Name:      name,
		Time:      time,
		SystemOut: logs,
		Result:    result,
	})
}

// TestCase contains information about an individual test case.
type TestCase struct {
	Name      string    `xml:"name,attr"`
	Time      time.Time `xml:"time,attr"`
	SystemOut string    `xml:"system-out"`
	Result    *Result   `xml:",omitempty"`
}

// Result represents the final state of the test case.
type Result struct {
	XMLName xml.Name
	Type    string `xml:"type,attr"`
	Message string `xml:",innerxml"`
}
