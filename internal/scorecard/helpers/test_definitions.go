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

package schelpers

import (
	"context"

	scapiv1alpha2 "github.com/operator-framework/operator-sdk/pkg/apis/scorecard/v1alpha2"
)

// Test provides methods for running scorecard tests
type Test interface {
	GetName() string
	GetDescription() string
	GetLabels() map[string]string
	Run(context.Context) *TestResult
}

// TestResult contains a test's points, suggestions, and errors
type TestResult struct {
	State       scapiv1alpha2.State
	Test        Test
	Suggestions []string
	Errors      []error
	Log         string
	CRName      string
}

// TestInfo contains information about the scorecard test
type TestInfo struct {
	Name        string
	Description string
	Labels      map[string]string
}

// GetName return the test name
func (i TestInfo) GetName() string { return i.Name }

// GetDescription returns the test description
func (i TestInfo) GetDescription() string { return i.Description }

// GetLabels returns the labels for this test
func (i TestInfo) GetLabels() map[string]string { return i.Labels }
