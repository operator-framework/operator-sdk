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

package v1alpha3

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigurationKind is the default scorecard componentconfig kind.
const ConfigurationKind = "Configuration"

// Configuration represents the set of test configurations which scorecard would run.
type Configuration struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`

	// Do not use metav1.ObjectMeta because this "object" should not be treated as an actual object.
	Metadata struct {
		// Name is a required field for kustomize-able manifests, and is not used on-cluster (nor is the config itself).
		Name string `json:"name,omitempty" yaml:"name,omitempty"`
	} `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Stages is a set of test stages to run. Once a stage is finished, the next stage in the slice will be run.
	Stages []StageConfiguration `json:"stages" yaml:"stages"`
}

// StageConfiguration configures a set of tests to be run.
type StageConfiguration struct {
	// Parallel, if true, will run each test in tests in parallel.
	// The default is to wait until a test finishes to run the next.
	Parallel bool `json:"parallel,omitempty" yaml:"parallel,omitempty"`
	// Tests are a list of tests to run.
	Tests []TestConfiguration `json:"tests" yaml:"tests"`
}

// TestConfiguration configures a specific scorecard test, identified by entrypoint.
type TestConfiguration struct {
	// Image is the name of the test image.
	Image string `json:"image" yaml:"image"`
	// Entrypoint is a list of commands and arguments passed to the test image.
	Entrypoint []string `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	// Labels further describe the test and enable selection.
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}
