// Copyright 2021 The Operator-SDK Authors
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

package v1alpha2

// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type Hog struct {
	// Should be in status but not spec, since Hog isn't in DummySpec
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="boss-hog-engine"
	Engine Engine `json:"engine"`
	// Not in spec or status, no boolean annotation
	// +operator-sdk:csv:customresourcedefinitions:displayName="doesnt-matter"
	Brand string `json:"brand"`
	// Not in spec or status
	Helmet string `json:"helmet"`
	// Fields should be inlined
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Inlined InlinedComponent `json:",inline"`
	// Fields should be inlined
	InlinedComponent `json:",inline"`
	// Should be ignored
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Ignored IgnoredComponent `json:"-"`
	// Should be ignored, but exported children should not be
	notExported `json:",inline"`
}

type notExported struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Public string `json:"foo"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +operator-sdk:csv:customresourcedefinitions:type=status
	private string `json:"-"`
}

// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type Engine struct {
	// Should not be included, no annotations.
	Pistons []string `json:"pistons"`
}

// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type Wheel struct {
	// Type should be in spec with path equal to wheels[0].type
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Wheel Type",xDescriptors="urn:alm:descriptor:com.tectonic.ui:arrayFieldGroup:wheels" ; "urn:alm:descriptor:com.tectonic.ui:text"
	Type string `json:"type"`
}

// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type InlinedComponent struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +operator-sdk:csv:customresourcedefinitions:type=status
	SeatMaterial string `json:"seatMaterial"`
}

// +k8s:deepcopy-gen=false
// +k8s:openapi-gen=false
type IgnoredComponent struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +operator-sdk:csv:customresourcedefinitions:type=status
	TrunkSpace string `json:"trunkSpace"`
}
