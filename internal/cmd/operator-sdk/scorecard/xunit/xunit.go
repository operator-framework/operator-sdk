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
	Name      string                `json:"name,omitempty"`
	Time      string                `json:"time,omitempty"`
	Classname string                `json:"classname,omitempty"`
	Group     string                `json:"group,omitempty"`
	Failures  []XUnitComplexFailure `json:"failure,omitempty"`
	Errors    []XUnitComplexError   `json:"error,omitempty"`
	Skipped   []XUnitComplexSkipped `json:"skipped,omitempty"`
}

// TestSuite contains for details about a test beyond the final status
type TestSuite struct {
	// Name is the name of the test
	Name       string      `json:"name,omitempty"`
	Tests      string      `json:"tests,omitempty"`
	Failures   string      `json:"failures,omitempty"`
	Errors     string      `json:"errors,omitempty"`
	Group      string      `json:"group,omitempty"`
	Skipped    string      `json:"skipped,omitempty"`
	Timestamp  string      `json:"timestamp,omitempty"`
	Hostname   string      `json:"hostnames,omitempty"`
	ID         string      `json:"id,omitempty"`
	Package    string      `json:"package,omitempty"`
	File       string      `json:"file,omitempty"`
	Log        string      `json:"log,omitempty"`
	URL        string      `json:"url,omitempty"`
	Version    string      `json:"version,omitempty"`
	TestSuites []TestSuite `json:"testsuite,omitempty"`
	TestCases  []TestCase  `json:"testcase,omitempty"`
}

// TestSuites is the top level object for amassing Xunit test results
type TestSuites struct {
	// Name is the name of the test
	Name      string      `json:"name,omitempty"`
	Tests     string      `json:"tests,omitempty"`
	Failures  string      `json:"failures,omitempty"`
	Errors    string      `json:"errors,omitempty"`
	TestSuite []TestSuite `json:"testsuite,omitempty"`
}

// XUnitComplexError contains a type header along with the error messages
type XUnitComplexError struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
}

// XUnitComplexFailure contains a type header along with the failure logs
type XUnitComplexFailure struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
}

// XUnitComplexSkipped contianers a type header along with associated run logs
type XUnitComplexSkipped struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
}
