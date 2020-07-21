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

// scorecardConfigKind is the default scorecard componentconfig kind.
const ScorecardConfigurationKind = "ScorecardConfiguration"

// ScorecardConfiguration is the Schema for the scorecardconfigurations API
type ScorecardConfiguration struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	// Do not use metav1.ObjectMeta because this object should not be apply-able.
	Metadata struct {
		Name string `json:"name,omitempty" yaml:"name,omitempty"`
	} `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec ScorecardConfigurationSpec `json:"spec,omitempty" yaml:"spec,omitempty"`
}

// ScorecardConfigurationSpec represents the set of test configurations which scorecard would run based on user input.
type ScorecardConfigurationSpec struct {
	Stages []StageConfiguration `json:"stages" yaml:"stages"`
}

type StageConfiguration struct {
	Parallel bool                `json:"parallel,omitempty" yaml:"parallel,omitempty"`
	Tests    []TestConfiguration `json:"tests" yaml:"tests"`
}

type TestConfiguration struct {
	// Image is the name of the testimage
	Image string `json:"image" yaml:"image"`
	// Entrypoint is list of commands and arguments passed to the test image
	Entrypoint []string `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	// Labels that further describe the test and enable selection
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}
