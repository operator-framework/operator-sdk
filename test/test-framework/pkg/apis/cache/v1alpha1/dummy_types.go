// Copyright 2018 The Operator-SDK Authors
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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NoKindSpec struct {
	// Not included in anything, no kind type
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Size int32 `json:"size"`
	// Not included in anything, no kind type
	Boss Hog `json:"hog"`
}
type NoKindStatus struct {
	// Not included in anything, no kind type
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Nodes []string `json:"nodes"`
}

type DummySpec struct {
	// Should be in spec
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="dummy-pods"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:podCount"
	Size int32 `json:"size"`
}
type DummyStatus struct {
	// Should be in status but not spec, since DummyStatus isn't in DummySpec
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Nodes []string `json:"nodes"`
	// Not included in status but children should be
	Boss Hog `json:"hog"`
}

type Hog struct {
	// Should be in status but not spec, since Hog isn't in DummySpec
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="boss-hog-engine"
	Engine Engine `json:"engine"`
	// Not in spec or status, no boolean annotation
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors.displayName="doesnt-matter"
	Brand string `json:"brand"`
	// Not in spec or status
	Helmet string `json:"helmet"`
}

type Engine struct {
	// Should not be included, no annotations.
	Pistons []string `json:"pistons"`
}

type OtherDummyStatus struct {
	// Should be in status but not spec, since this isn't a spec type
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.statusDescriptors=true
	Nothing string `json:"nothing"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Dummy is the Schema for the dummy API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=memcacheds,scope=Namespaced
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Dummy App"
// +operator-sdk:gen-csv:customresourcedefinitions.resources="Deployment,v1,\"dummy-deployment\""
// +operator-sdk:gen-csv:customresourcedefinitions.resources="ReplicaSet,v1beta2,\"dummy-replicaset\""
// +operator-sdk:gen-csv:customresourcedefinitions.resources="Pod,v1,\"dummy-pod\""
type Dummy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DummySpec   `json:"spec,omitempty"`
	Status DummyStatus `json:"status,omitempty"`
}

// OtherDummy is the Schema for the other dummy API
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Other Dummy App"
// +operator-sdk:gen-csv:customresourcedefinitions.resources="Service,v1,\"other-dummy-service\""
// +operator-sdk:gen-csv:customresourcedefinitions.resources="Pod,v1,\"other-dummy-pod\""
type OtherDummy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Hog              `json:"spec,omitempty"`
	Status OtherDummyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DummyList contains a list of Dummy
type DummyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dummy `json:"items"`
}
