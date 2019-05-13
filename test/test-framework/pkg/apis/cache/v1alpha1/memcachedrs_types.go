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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MemcachedRSSpec defines the desired state of MemcachedRS
// +k8s:openapi-gen=true
type MemcachedRSSpec struct {
	NumNodes int32 `json:"numNodes"`
}

// MemcachedRSStatus defines the observed state of MemcachedRS
// +k8s:openapi-gen=true
type MemcachedRSStatus struct {
	NodeList []string `json:"nodeList"`
	Test     bool     `json:"test"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MemcachedRS is the Schema for the memcachedrs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type MemcachedRS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemcachedRSSpec   `json:"spec,omitempty"`
	Status MemcachedRSStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MemcachedRSList contains a list of MemcachedRS
type MemcachedRSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MemcachedRS `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MemcachedRS{}, &MemcachedRSList{})
}
