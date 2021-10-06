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

// TestCase contain the core information from a test run, including its name and status
type TestCase struct {
	// Name is the name of the test
	Name      string                `xml:"name,attr,omitempty"`
	Time      string                `xml:"time,attr,omitempty"`
	Classname string                `xml:"classname,attr,omitempty"`
	Group     string                `xml:"group,attr,omitempty"`
	Failures  []XUnitComplexFailure `xml:"failure,omitempty"`
	Errors    []XUnitComplexError   `xml:"error,omitempty"`
	Skipped   []XUnitComplexSkipped `xml:"skipped,omitempty"`
}

// TestSuite contains for details about a test beyond the final status
type TestSuite struct {
	// Name is the name of the test
	Name       string      `xml:"name,attr,omitempty"`
	Tests      string      `xml:"tests,attr,omitempty"`
	Failures   string      `xml:"failures,attr,omitempty"`
	Errors     string      `xml:"errors,attr,omitempty"`
	Group      string      `xml:"group,attr,omitempty"`
	Skipped    string      `xml:"skipped,attr,omitempty"`
	Timestamp  string      `xml:"timestamp,attr,omitempty"`
	Hostname   string      `xml:"hostnames,attr,omitempty"`
	ID         string      `xml:"id,attr,omitempty"`
	Package    string      `xml:"package,attr,omitempty"`
	File       string      `xml:"file,attr,omitempty"`
	Log        string      `xml:"log,attr,omitempty"`
	URL        string      `xml:"url,attr,omitempty"`
	Version    string      `xml:"version,attr,omitempty"`
	TestSuites []TestSuite `xml:"testsuite,omitempty"`
	TestCases  []TestCase  `xml:"testcase,omitempty"`
}

// TestSuites is the top level object for amassing Xunit test results
type TestSuites struct {
	// Name is the name of the test
	Name      string      `xml:"name,attr,omitempty"`
	Tests     string      `xml:"tests,attr,omitempty"`
	Failures  string      `xml:"failures,attr,omitempty"`
	Errors    string      `xml:"errors,attr,omitempty"`
	TestSuite []TestSuite `xml:"testsuite,omitempty"`
}

// XUnitComplexError contains a type header along with the error messages
type XUnitComplexError struct {
	Type    string `xml:"type,attr,omitempty"`
	Message string `xml:"message,attr,omitempty"`
}

// XUnitComplexFailure contains a type header along with the failure logs
type XUnitComplexFailure struct {
	Type    string `xml:"type,attr,omitempty"`
	Message string `xml:"message,attr,omitempty"`
}

// XUnitComplexSkipped contains a type header along with associated run logs
type XUnitComplexSkipped struct {
	Type    string `xml:"type,attr,omitempty"`
	Message string `xml:"message,attr,omitempty"`
}
