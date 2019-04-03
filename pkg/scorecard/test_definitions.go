// Copyright 2019 The Operator-SDK Authors
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

package scorecard

import (
	"context"
)

// Type Definitions

// Test provides methods for running scorecard tests
type Test interface {
	GetName() string
	GetDescription() string
	IsCumulative() bool
	Run(context.Context) *TestResult
}

// TestResult contains a test's points, suggestions, and errors
type TestResult struct {
	Test          Test
	EarnedPoints  int
	MaximumPoints int
	Suggestions   []string
	Errors        []error
}

// TestInfo contains information about the scorecard test
type TestInfo struct {
	Name        string
	Description string
	// If a test is set to cumulative, the scores of multiple runs of the same test on separate CRs are added together for the total score.
	// If cumulative is false, if any test failed, the total score is 0/1. Otherwise 1/1.
	Cumulative bool
}

// GetName return the test name
func (i TestInfo) GetName() string { return i.Name }

// GetDescription returns the test description
func (i TestInfo) GetDescription() string { return i.Description }

// IsCumulative returns true if the test's scores are intended to be cumulative
func (i TestInfo) IsCumulative() bool { return i.Cumulative }

// TestSuite contains a list of tests and results, along with the relative weights of each test
type TestSuite struct {
	TestInfo
	Tests       []Test
	TestResults []*TestResult
	Weights     map[string]float64
}

// Helper functions

// AddTest adds a new Test to a TestSuite along with a relative weight for the new Test
func (ts *TestSuite) AddTest(t Test, weight float64) {
	ts.Tests = append(ts.Tests, t)
	ts.Weights[t.GetName()] = weight
}

// TotalScore calculates and returns the total score of all run Tests in a TestSuite
func (ts *TestSuite) TotalScore() (score int) {
	floatScore := 0.0
	for _, result := range ts.TestResults {
		if result.MaximumPoints != 0 {
			floatScore += (float64(result.EarnedPoints) / float64(result.MaximumPoints)) * ts.Weights[result.Test.GetName()]
		}
	}
	// scale to a percentage
	addedWeights := 0.0
	for _, weight := range ts.Weights {
		addedWeights += weight
	}
	floatScore = floatScore * (100 / addedWeights)
	return int(floatScore)
}

// Run runs all Tests in a TestSuite
func (ts *TestSuite) Run(ctx context.Context) {
	for _, test := range ts.Tests {
		ts.TestResults = append(ts.TestResults, test.Run(ctx))
	}
}

// NewTestSuite returns a new TestSuite with a given name and description
func NewTestSuite(name, description string) *TestSuite {
	return &TestSuite{
		TestInfo: TestInfo{
			Name:        name,
			Description: description,
		},
		Weights: make(map[string]float64),
	}
}
